// Copyright(C) 2019 github.com/hidu  All Rights Reserved.
// Author: hidu (duv123+git@baidu.com)
// Date: 2019/12/21

package gofmt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sync"

	"github.com/fsgo/go_fmt/internal/common"
	"github.com/fsgo/go_fmt/internal/xpasser"
)

// NewFormatter 创建一个新的带默认格式化规则的格式化实例
func NewFormatter() *Formatter {
	return &Formatter{}
}

// Formatter 代码格式化实例
type Formatter struct {
	// PrintResult 用于格式化过程中，打印结果
	PrintResult func(fileName string, change bool, err error)
}

// Execute 执行代码格式化
func (ft *Formatter) Execute(opt *Options) error {
	opt = opt.Clone()
	if e := opt.Check(); e != nil {
		return e
	}
	return ft.execute(opt)
}

// OnExecute hook Before Execute
var OnExecute = func(opt *Options) error {
	return nil
}

func (ft *Formatter) execute(opt *Options) error {
	if err := OnExecute(opt); err != nil {
		return err
	}
	files, err := opt.AllGoFiles()
	if err != nil {
		return err
	}
	// Deduplicate paths before workers mutate syntax trees or write files.
	seen := make(map[string]bool, len(files))
	unique := files[:0]
	for _, file := range files {
		key := file
		if file != common.NameSTDIN {
			key, err = filepath.Abs(file)
			if err != nil {
				return err
			}
			key, err = filepath.EvalSymlinks(key)
			if err != nil {
				return err
			}
		}
		if !seen[key] {
			unique = append(unique, file)
			seen[key] = true
		}
	}
	files = unique
	slices.Sort(files)
	pr := xpasser.NewProgram()
	if !opt.FormatOnly && len(files) > 0 && files[0] != common.NameSTDIN {
		if err := pr.LoadFiles(*opt, files, nil); err != nil {
			return fmt.Errorf("load source packages: %w", err)
		}
	}
	var diffs diffResults
	type result struct {
		origin, pretty []byte
		formatted      bool
		err            error
	}
	results := make([]result, len(files))
	var wg sync.WaitGroup
	limit := make(chan struct{}, max(1, runtime.GOMAXPROCS(0)))
	for i, fileName := range files {
		limit <- struct{}{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-limit }()
			if fileName == common.NameSTDIN {
				results[i].origin, results[i].pretty, results[i].formatted, results[i].err = ft.Format(
					fileName,
					nil,
					opt,
				)
				return
			}
			src, e := os.ReadFile(fileName)
			results[i].origin = src
			if e != nil {
				results[i].err = e
				return
			}
			results[i].pretty, results[i].formatted, results[i].err = formatFile(fileName, src, opt, pr)
		}()
	}
	wg.Wait()
	var changeNum int
	for i, result := range results {
		if diff := fileDiff(opt, files[i], result.origin, result.pretty, result.err); diff != nil {
			diffs = append(diffs, diff)
		}
		changed, e := ft.output(opt, files[i], result.origin, result.pretty, result.formatted, result.err)
		if e != nil {
			err = e
		}
		if changed {
			changeNum++
		}
	}

	if len(diffs) > 0 {
		diffs.Output(opt.DisplayFormat)
	}
	if err != nil {
		return err
	}
	if opt.DisplayDiff && changeNum > 0 {
		return fmt.Errorf("%d files should be formatted", changeNum)
	}
	return nil
}

func (ft *Formatter) printFmtResult(fileName string, change bool, event string, color common.ConsoleColor, err error) {
	if ft.PrintResult != nil {
		ft.PrintResult(fileName, change, err)
		return
	}
	txt := fmt.Sprintf(" %8s : %s", event, fileName)
	if err != nil {
		txt += " " + err.Error()
	}
	txt = color(txt)
	fmt.Fprint(os.Stderr, txt, "\n")
}

func fileDiff(opt *Options, fileName string, originSrc []byte, prettySrc []byte, err error) *diffResult {
	if bytes.Equal(originSrc, prettySrc) || !opt.DisplayDiff || err != nil {
		return nil
	}
	result := common.Diff(string(originSrc), string(prettySrc), opt.Trace)
	if result == nil {
		return nil
	}

	df := &diffResult{
		File:   fileName,
		Diffs:  result.Detail(),
		result: result,
	}
	return df
}

func (ft *Formatter) output(
	opt *Options,
	fileName string,
	originSrc, prettySrc []byte,
	formatted bool,
	err error,
) (bool, error) {
	changed := !bytes.Equal(originSrc, prettySrc)
	if err != nil {
		ft.printFmtResult(fileName, true, "error", common.ConsoleRed, err)
		return changed, err
	}

	if !opt.Write && !opt.DisplayDiff {
		fmt.Print(string(prettySrc))
		return changed, nil
	}

	if !changed {
		if formatted {
			ft.printFmtResult(fileName, false, "pretty", common.ConsoleGreen, nil)
		} else {
			ft.printFmtResult(fileName, false, "skipped", common.ConsoleGrey, nil)
		}
		return changed, nil
	}

	if opt.Write {
		err = os.WriteFile(fileName, prettySrc, 0)
		ft.printFmtResult(fileName, true, "rewrote", common.ConsoleRed, err)
		return changed, err
	} else if opt.DisplayDiff {
		ft.printFmtResult(fileName, true, "ugly", common.ConsoleRed, nil)
	}

	return changed, nil
}

// Format 格式化文件，获取格式化后的内容
func (ft *Formatter) Format(
	fileName string,
	src []byte,
	opt *Options,
) (origin []byte, pretty []byte, formatted bool, err error) {
	if src == nil {
		if fileName == common.NameSTDIN {
			src, err = io.ReadAll(os.Stdin)
		} else {
			src, err = os.ReadFile(fileName)
		}
		if err != nil {
			return nil, nil, false, err
		}
	}
	sourceName := fileName
	if fileName == common.NameSTDIN {
		sourceName = opt.SourcePath
		if sourceName == "" {
			sourceName = "stdin.go"
		}
	}
	sourceName, err = filepath.Abs(sourceName)
	if err != nil {
		return nil, nil, false, err
	}
	pretty, formatted, err = Format(sourceName, src, opt)
	if err != nil {
		return nil, nil, false, err
	}
	return src, pretty, formatted, nil
}

// FormatAndWriteFile 格式化并写入文件
func (ft *Formatter) FormatAndWriteFile(fileName string, opt *Options) (bool, error) {
	originSrc, prettySrc, _, err := ft.Format(fileName, nil, opt)
	if err != nil {
		return false, err
	}

	if bytes.Equal(originSrc, prettySrc) {
		return false, nil
	}

	if opt.Write {
		err = os.WriteFile(fileName, prettySrc, 0)
	}
	return true, err
}

type diffResult struct {
	Diffs  any
	result common.DiffResult
	File   string
	Error  string
}

type diffResults []*diffResult

func (drs diffResults) Output(format string) {
	if format == "json" {
		bf, _ := json.MarshalIndent(drs, " ", "    ")
		fmt.Fprint(os.Stdout, string(bf), "\n")
		return
	}

	for i := 0; i < len(drs); i++ {
		item := drs[i]
		title := common.ConsoleRed(item.File)
		msg := item.result.String()
		if len(item.Error) > 0 {
			msg += "Error:" + common.ConsoleRed(item.Error)
		}
		fmt.Fprint(os.Stdout, title, "\n", msg, "\n")
	}
}
