package gofmt

import (
	"errors"
	"fmt"
	"go/ast"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/daixiang0/gci/pkg/config"
	"github.com/daixiang0/gci/pkg/gci"
	gcilog "github.com/daixiang0/gci/pkg/log"
	"github.com/daixiang0/gci/pkg/parse"
	"github.com/daixiang0/gci/pkg/section"
	"github.com/daixiang0/gci/pkg/specificity"
	"github.com/golangci/golines/shorten"
	"golang.org/x/mod/modfile"
	"golang.org/x/tools/imports"
	fumpt "mvdan.cc/gofumpt/format"

	"github.com/fsgo/go_fmt/internal/common"
	"github.com/fsgo/go_fmt/internal/ximports"
)

// formatLayout implements the single gorgeous layout pipeline. Package repair
// follows semantic rewrites; grouping follows layout so the repository's order wins.
func formatLayout(req *common.Request, src []byte) ([]byte, error) {
	// Merge declarations before import repair to keep inline comments attached.
	var declarations int
	for _, decl := range req.AstFile.Decls {
		if gen, ok := decl.(*ast.GenDecl); ok && gen.Tok == token.IMPORT {
			declarations++
		}
	}
	if declarations > 1 && req.Opt.MergeImports {
		var err error
		src, err = ximports.FormatImports(req)
		if err != nil {
			return nil, err
		}
	}
	src, err := imports.Process(req.FileName, src, &imports.Options{Comments: true, TabIndent: true, TabWidth: 8})
	if err != nil {
		return nil, fmt.Errorf("repair imports: %w", err)
	}
	src, err = fumpt.Source(
		src,
		fumpt.Options{LangVersion: "go" + req.GoVersion(), ModulePath: req.Opt.LocalModule},
	)
	if err != nil {
		return nil, err
	}
	if req.Opt.MaxLineLength > 0 {
		cfg := shorten.NewDefaultConfig()
		cfg.MaxLen = req.Opt.MaxLineLength
		cfg.TabLen = 8
		cfg.ReformatTags = false
		src, err = shorten.NewShortener(cfg).Process(src)
		if err != nil {
			return nil, fmt.Errorf("shorten lines: %w", err)
		}
	}
	req.FSet, req.AstFile, err = common.ParseOneFile(req.FileName, src)
	if err != nil {
		return nil, err
	}
	var localPaths []string
	var excluded common.Modules
	if mod, err := common.FindGoModPath(req.FileName); err == nil {
		localPaths, err = localModulePaths(req.FileName)
		if err != nil {
			return nil, err
		}
		nested, err := common.ListModules(filepath.Dir(mod))
		if err != nil {
			return nil, err
		}
		for _, module := range nested {
			if !common.SliceHas(localPaths, module) {
				excluded = append(excluded, module)
			}
		}
	}
	// -local overrides the inferred module, including workspace membership.
	if req.Opt.LocalModule != "" && !strings.HasPrefix(req.Opt.LocalModule, "_") {
		if len(localPaths) == 0 || !common.SliceHas(localPaths, req.Opt.LocalModule) {
			localPaths = []string{req.Opt.LocalModule}
		}
	}
	// GCI does not retain standalone comments inside import declarations.
	// Keep the comment-aware gorgeous sorter for these blocks and explicit policies.
	if !req.Opt.MergeImports || req.Opt.ImportGroupFn != nil || len(req.Opt.ThirdModules) > 0 ||
		standaloneImportComments(req.AstFile) {
		if req.Opt.ImportGroupFn == nil {
			thirdModules := append(common.Modules(nil), req.Opt.ThirdModules...)
			thirdModules = append(thirdModules, excluded...)
			req.Opt.ThirdModules = thirdModules
			req.Opt.ImportGroupFn = func(path string, opt common.Options) int {
				if !thirdModules.PkgIn(path) && common.Modules(localPaths).PkgIn(path) {
					return opt.GetImportGroup(common.ImportGroupCurrentModule)
				}
				return ximports.DefaultImportGroup(path, opt)
			}
		}
		return ximports.FormatImports(req)
	}
	order := req.Opt.ImportGroupRule
	if order == "" {
		order = "stc"
	}
	var sections section.SectionList
	for _, group := range order {
		switch group {
		case 's':
			sections = append(sections, &section.Standard{})
		case 't':
			sections = append(sections, &section.Default{})
		case 'c':
			if len(localPaths) > 0 {
				sections = append(
					sections,
					&moduleSection{
						LocalModule: section.LocalModule{Paths: localPaths},
						excluded:    excluded,
					},
				)
			}
		}
	}
	gcilog.InitLogger()
	cfg := config.Config{
		BoolConfig:        config.BoolConfig{CustomOrder: true},
		Sections:          sections,
		SectionSeparators: section.DefaultSectionSeparators(),
	}
	_, src, err = gci.LoadFormat(src, req.FileName, cfg)
	if err != nil {
		return nil, fmt.Errorf("group imports: %w", err)
	}
	return src, nil
}

// moduleSection keeps nested independent modules in the third-party group.
// Modules explicitly listed in go.work remain local.
type moduleSection struct {
	section.LocalModule
	excluded common.Modules
}

func (s *moduleSection) MatchSpecificity(spec *parse.GciImports) specificity.MatchSpecificity {
	if s.excluded.PkgIn(spec.Path) {
		return specificity.MisMatch{}
	}
	return s.LocalModule.MatchSpecificity(spec)
}

// standaloneImportComments identifies import comments that GCI cannot associate
// with a spec. Comments attached to a spec are handled by GCI itself.
func standaloneImportComments(file *ast.File) bool {
	attached := make(map[*ast.CommentGroup]bool)
	for _, spec := range file.Imports {
		attached[spec.Doc] = true
		attached[spec.Comment] = true
	}
	for _, decl := range file.Decls {
		gen, ok := decl.(*ast.GenDecl)
		if !ok || gen.Tok != token.IMPORT {
			continue
		}
		for _, comment := range file.Comments {
			if comment.Pos() > gen.Pos() && comment.End() < gen.End() && !attached[comment] {
				return true
			}
		}
	}
	return false
}

// Resolve workspace modules relative to the source, not the process's cwd.
func localModulePaths(filename string) ([]string, error) {
	path, err := filepath.Abs(filename)
	if err != nil {
		return nil, err
	}
	work := os.Getenv("GOWORK")
	if work == "" {
		for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
			candidate := filepath.Join(dir, "go.work")
			if _, err := os.Stat(candidate); err == nil {
				work = candidate
				break
			} else if !errors.Is(err, fs.ErrNotExist) {
				return nil, err
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	if work != "" && work != "off" {
		data, err := os.ReadFile(work)
		if err != nil {
			return nil, err
		}
		wf, err := modfile.ParseWork(work, data, nil)
		if err != nil {
			return nil, err
		}
		var paths []string
		for _, use := range wf.Use {
			dir := use.Path
			if !filepath.IsAbs(dir) {
				dir = filepath.Join(filepath.Dir(work), dir)
			}
			module, err := common.ModuleByFile(filepath.Join(dir, "go.mod"))
			if err != nil {
				return nil, err
			}
			paths = append(paths, module)
		}
		if len(paths) > 0 {
			return paths, nil
		}
		return nil, fmt.Errorf("no modules in %s", work)
	}
	mod, err := common.FindGoModPath(path)
	if err != nil {
		return nil, fmt.Errorf("localmodule section: %w", err)
	}
	module, err := common.ModuleByFile(mod)
	if err != nil {
		return nil, err
	}
	return []string{strings.TrimSpace(module)}, nil
}
