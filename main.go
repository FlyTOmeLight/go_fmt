// Copyright(C) 2019 github.com/hidu  All Rights Reserved.
// Author: hidu (duv123+git@baidu.com)
// Date: 2019/12/16

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"

	"github.com/fsgo/go_fmt/gofmtapi"
	"github.com/fsgo/go_fmt/internal/common"
	"github.com/fsgo/go_fmt/internal/gofmt"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "fix" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
		defer cancel()
		if err := gofmt.Modernize(ctx, os.Args[2:]); err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				os.Exit(exit.ExitCode())
			}
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		return
	}

	gf := gofmtapi.NewFormatter()
	opt := gofmtapi.NewOptions()
	opt.BindFlags()

	err := gf.Execute(opt)
	if err != nil {
		fmt.Fprint(os.Stderr, common.ConsoleRed(err.Error())+"\n")
		os.Exit(2)
	}
}
