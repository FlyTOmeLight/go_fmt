// Copyright(C) 2019 github.com/hidu  All Rights Reserved.
// Author: hidu (duv123+git@baidu.com)
// Date: 2019/12/16

package gofmt

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/fsgo/go_fmt/internal/common"
	"github.com/fsgo/go_fmt/internal/localmodule"
	"github.com/fsgo/go_fmt/internal/simplify"
	"github.com/fsgo/go_fmt/internal/xanalysis"
	"github.com/fsgo/go_fmt/internal/xast"
	"github.com/fsgo/go_fmt/internal/xpasser"
)

// Options 别名
type Options = common.Options

// Format 输出格式化的 go 代码
func Format(fileName string, src []byte, opts *Options) (code []byte, formatted bool, err error) {
	opts = opts.Clone()
	if err := opts.Check(); err != nil {
		return nil, false, err
	}
	if src == nil {
		src, err = os.ReadFile(fileName)
		if err != nil {
			return nil, false, err
		}
	}
	pr := xpasser.NewProgram()
	if !opts.FormatOnly && !common.DoNotEdit(fileName, src) {
		ap, err := filepath.Abs(fileName)
		if err != nil {
			return nil, false, err
		}
		if err := pr.LoadFiles(*opts, []string{fileName}, map[string][]byte{ap: src}); err != nil {
			return nil, false, fmt.Errorf("load source packages: %w", err)
		}
	}
	return formatFile(fileName, src, opts, pr)
}

func formatFile(
	fileName string,
	src []byte,
	opts *Options,
	pr *xpasser.Program,
) (code []byte, formatted bool, err error) {
	defer func() {
		if re := recover(); re != nil {
			code = nil
			formatted = false
			err = fmt.Errorf("format %q: %v", fileName, re)
		}
	}()
	options := opts.Clone()
	if src == nil {
		src, err = os.ReadFile(fileName)
		if err != nil {
			return nil, false, err
		}
	}

	if common.DoNotEdit(fileName, src) {
		return src, false, nil
	}

	module, err := localmodule.Get(*options, fileName)
	if err != nil {
		return nil, false, err
	}

	if options.Trace {
		log.Println("[go.module]", fileName, "--->", module)
	}

	options.LocalModule = module

	file, err := pr.ParserFile(fileName, src)
	if err != nil {
		return nil, false, err
	}

	req := &common.Request{
		FileName:  fileName,
		FSet:      pr.FSet,
		AstFile:   file,
		TypesInfo: pr.TypesInfo(fileName),
		Opt:       *options,
	}
	// ast.Print(fileSet, file)

	code, err = fix(req)
	if err != nil {
		return nil, false, err
	}
	return code, true, err
}

func fix(req *common.Request) ([]byte, error) {
	if !req.Opt.FormatOnly {
		xanalysis.Format(req)
	}

	// ast.Print(fileSet,file)
	if req.Opt.Simplify && !req.Opt.FormatOnly {
		simplify.Format(req)
	}
	FormatComments(req)

	if !req.Opt.FormatOnly && len(req.Opt.RewriteRules) > 0 {
		err := simplify.Rewrites(req, req.Opt.RewriteRules)
		if err != nil {
			return nil, fmt.Errorf("rewrite failed: %w", err)
		}
	} else if !req.Opt.FormatOnly && req.Opt.RewriteWithBuildIn {
		// 当传入 rewrite 规则的时候，不使用默认规则
		err := simplify.Rewrites(req, simplify.BuildInRewriteRules())
		if err != nil {
			return nil, fmt.Errorf("rewrite with build in rules failed: %w", err)
		}
	}

	// if opt.FieldAlignment == 1 {
	// 	fieldalignment.Run(fset, file, true)
	// }

	if req.Opt.Extra {
		xast.Format(req)
	}

	code, err := req.FormatFile()
	if err != nil {
		return nil, err
	}
	return formatLayout(req, code)
}
