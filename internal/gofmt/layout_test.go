package gofmt

import (
	"bytes"
	"errors"
	"fmt"
	"go/format"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fsgo/go_fmt/internal/common"
)

func writeSource(t *testing.T, dir, name, src string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func testModule(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	writeSource(t, dir, "go.mod", "module example.com/sample\n\ngo 1.26.0\n")
	return dir
}

func TestDefaultFormatting(t *testing.T) {
	tests := []struct {
		name, src, contains, absent string
	}{
		{
			name:     "strict layout",
			src:      "package p\n\nfunc f() {\n\nprintln(1)\n}\n",
			contains: "func f() {\n\tprintln(1)\n}",
			absent:   "",
		},
		{
			name:     "import repair",
			src:      "package p\nimport \"os\"\nfunc f(){fmt.Println(1)}\n",
			contains: "\"fmt\"",
			absent:   "\"os\"",
		},
		{
			name:     "import groups",
			src:      "package p\nimport(\"example.com/sample/local\";\"fmt\";\"third.example/pkg\")\nvar _, _, _ = local.F, fmt.Println, pkg.F\n",
			contains: "\"fmt\"\n\n\t\"third.example/pkg\"\n\n\t\"example.com/sample/local\"",
			absent:   "",
		},
		{
			name:     "long line",
			src:      "package p\nfunc f(){\nprintln(\"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\",\"bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\",\"cccccccccccccccccccccccccccccccccccccccc\")\n}\n",
			contains: "println(\n",
			absent:   "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			opt := common.NewDefaultOptions()
			name := filepath.Join(testModule(t), "sample.go")
			out, _, err := Format(name, []byte(tt.src), opt)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(out), tt.contains) ||
				tt.absent != "" && strings.Contains(string(out), tt.absent) {
				t.Fatalf("expected %q without %q:\n%s", tt.contains, tt.absent, out)
			}
			again, _, err := Format(name, out, opt)
			if err != nil || !bytes.Equal(out, again) {
				t.Fatalf("not idempotent: %v\nfirst:\n%s\nsecond:\n%s", err, out, again)
			}
			canonical, err := format.Source(out)
			if err != nil || !bytes.Equal(canonical, out) {
				t.Fatalf("not gofmt compatible: %v", err)
			}
		})
	}
}

func TestWorkspaceImportGroups(t *testing.T) {
	t.Setenv("GOWORK", "")
	dir := t.TempDir()
	writeSource(t, dir, "go.work", "go 1.26.0\nuse (\n./app\n./lib\n)\n")
	writeSource(t, dir, "app/go.mod", "module example.com/app\n\ngo 1.26.0\n")
	writeSource(t, dir, "lib/go.mod", "module example.com/lib\n\ngo 1.26.0\n")
	src := []byte(
		"package p\nimport (\"example.com/lib/foo\";\"fmt\";\"other.example/pkg\")\nvar _, _, _ = foo.F, fmt.Println, pkg.F\n",
	)
	opt := common.NewDefaultOptions()
	opt.FormatOnly = true
	out, _, err := Format(filepath.Join(dir, "app/sub/file.go"), src, opt)
	if err != nil {
		t.Fatal(err)
	}
	want := "\"other.example/pkg\"\n\n\t\"example.com/lib/foo\""
	if !strings.Contains(string(out), want) {
		t.Fatalf("workspace group missing:\n%s", out)
	}
}

func TestCheckDoesNotModifyModule(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := testModule(t)
	mod := "module example.com/sample\n\ngo 1.26.0\nrequire example.com/unused v0.0.0\nreplace example.com/unused => ./unused\n"
	writeSource(t, dir, "go.mod", mod)
	writeSource(t, dir, "unused/go.mod", "module example.com/unused\n\ngo 1.26.0\n")
	file := writeSource(t, dir, "sample.go", "package sample\nvar A=1\n")
	opt := &Options{DisplayDiff: true, TabIndent: true, Files: []string{file}, LocalModule: "auto"}
	ft := NewFormatter()
	ft.PrintResult = func(string, bool, error) {}
	if err := ft.Execute(opt); err == nil {
		t.Fatal("expected a formatting difference")
	}
	got, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil || string(got) != mod {
		t.Fatalf("module modified: %v\n%s", err, got)
	}
	if _, err := os.Stat(filepath.Join(dir, "go.sum")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("unexpected go.sum: %v", err)
	}
	got, err = os.ReadFile(file)
	if err != nil || string(got) != "package sample\nvar A=1\n" {
		t.Fatal("source modified by -d")
	}
}

func TestParallelResultsAreOrdered(t *testing.T) {
	dir := testModule(t)
	var files []string
	for i := 25; i >= 0; i-- {
		files = append(
			files,
			writeSource(t, dir, fmt.Sprintf("file%02d.go", i), fmt.Sprintf("package p\nvar A%d=1\n", i)),
		)
	}
	opt := &Options{FormatOnly: true, Write: true, TabIndent: true, LocalModule: "auto", Files: files}
	opt.Files = append(opt.Files, files[0])
	var order []string
	ft := NewFormatter()
	ft.PrintResult = func(name string, changed bool, err error) {
		if err != nil {
			t.Error(err)
		}
		order = append(order, filepath.Base(name))
	}
	if err := ft.Execute(opt); err != nil {
		t.Fatal(err)
	}
	if len(order) != 26 {
		t.Fatalf("files processed more than once: %v", order)
	}
	for i, name := range order {
		if name != fmt.Sprintf("file%02d.go", i) {
			t.Fatalf("nondeterministic order: %v", order)
		}
	}
}

func TestGeneratedFileIsPreserved(t *testing.T) {
	src := []byte("//go:build linux\n\n// Code generated by sample. DO NOT EDIT.\n\npackage p\nvar A=1\n")
	opt := &Options{FormatOnly: true, MaxLineLength: 30}
	out, formatted, err := Format(filepath.Join(t.TempDir(), "generated.go"), src, opt)
	if err != nil || formatted || !bytes.Equal(out, src) {
		t.Fatalf("generated source changed: %v\n%s", err, out)
	}
}

func TestImportCommentsArePreserved(t *testing.T) {
	src := []byte(
		"//go:build linux\n\npackage p\nimport (\n// module comment\n\"example.com/sample/x\" // module inline\n\"fmt\" // std inline\n)\nvar _, _ = x.F, fmt.Println\n",
	)
	opt := &Options{FormatOnly: true, MergeImports: true, LocalModule: "auto"}
	name := filepath.Join(testModule(t), "sample.go")
	out, _, err := Format(name, src, opt)
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{"//go:build linux", "// module comment", "// module inline", "// std inline"} {
		if bytes.Count(out, []byte(text)) != 1 {
			t.Fatalf("comment lost or duplicated: %s\n%s", text, out)
		}
	}
	again, _, err := Format(name, out, opt)
	if err != nil || !bytes.Equal(out, again) {
		t.Fatalf("comments not stable: %v\n%s\n%s", err, out, again)
	}
}

func TestRewriteTypeIdentity(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := testModule(t)
	src := `package sample
import "strings"
import "time"
var _ = strings.TrimSpace
type Clock struct{}
func (Clock) Sub(time.Time) int { return 1 }
func f(c Clock) int { return c.Sub(time.Now()) }
func g() string {
 strings := struct{Replace func(string,string,string,int) string}{Replace: func(s,a,b string,n int)string{return s}}
 return strings.Replace("a","a","b",-1)
}
func h(t time.Time) time.Duration { return t.Sub(time.Now()) }
`
	file := writeSource(t, dir, "sample.go", src)
	opt := &Options{Simplify: true, TabIndent: true, LocalModule: "auto"}
	out, _, err := Format(file, []byte(src), opt)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"c.Sub(time.Now())", "strings.Replace(", "time.Until(t)"} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	writeSource(t, dir, "sample.go", string(out))
	cmd := exec.Command("go", "test", "-mod=readonly", "./...")
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("rewritten source does not compile: %v\n%s", err, output)
	}
}

func TestIncompleteCallDoesNotPanic(t *testing.T) {
	dir := testModule(t)
	for _, call := range []string{`fmt.Sprintf()`, `fmt.Sprintf("%d")`} {
		src := []byte("package sample\nimport \"fmt\"\nvar _ = " + call + "\n")
		out, _, err := Format(filepath.Join(dir, "sample.go"), src, &Options{Simplify: true, TabIndent: true})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(out), call) {
			t.Fatalf("incomplete call changed:\n%s", out)
		}
	}
}

func TestStandaloneImportComments(t *testing.T) {
	opt := common.NewDefaultOptions()
	opt.FormatOnly = true
	name := filepath.Join(testModule(t), "sample.go")
	src := []byte(`package p
import (
 "fmt"
 // keep trailing
)
import "strings" // keep inline
func f() { fmt.Println(strings.TrimSpace(" x ")) }
`)
	out, _, err := Format(name, src, opt)
	if err != nil {
		t.Fatal(err)
	}
	for _, comment := range []string{"// keep trailing", "// keep inline"} {
		if bytes.Count(out, []byte(comment)) != 1 {
			t.Fatalf("comment lost or duplicated: %s\n%s", comment, out)
		}
	}
	again, _, err := Format(name, out, opt)
	if err != nil || !bytes.Equal(out, again) {
		t.Fatalf("comment layout unstable: %v\n%s\n%s", err, out, again)
	}
}

func TestDefaultModernization(t *testing.T) {
	opt := common.NewDefaultOptions()
	name := filepath.Join(testModule(t), "sample.go")
	src := []byte(
		"package p\nimport \"io/ioutil\"\nfunc f() interface{} { b, _ := ioutil.ReadFile(\"x\"); return b }\n",
	)
	out, _, err := Format(name, src, opt)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"f() any", "os.ReadFile", "\"os\""} {
		if !strings.Contains(string(out), want) {
			t.Fatalf("missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(string(out), "io/ioutil") {
		t.Fatalf("obsolete import retained:\n%s", out)
	}
}

func TestNestedModuleImportGroups(t *testing.T) {
	t.Setenv("GOWORK", "off")
	dir := testModule(t)
	writeSource(t, dir, "child/go.mod", "module example.com/sample/child\n\ngo 1.26.0\n")
	opt := common.NewDefaultOptions()
	opt.FormatOnly = true
	src := []byte(
		"package p\nimport (\"example.com/sample/child\"; \"example.com/sample/local\"; \"fmt\")\nvar _, _, _ = child.F, local.F, fmt.Println\n",
	)
	out, _, err := Format(filepath.Join(dir, "sample.go"), src, opt)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(out), "\"example.com/sample/child\"\n\n\t\"example.com/sample/local\"") {
		t.Fatalf("independent nested module grouped as local:\n%s", out)
	}
}
