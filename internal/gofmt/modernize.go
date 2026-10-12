package gofmt

import (
	"context"
	"os"
	"os/exec"
)

// Modernize runs the installed Go toolchain's modernizers. Arguments and exit
// errors retain go fix semantics, including -diff and per-analyzer flags.
func Modernize(ctx context.Context, args []string) error {
	cmd := exec.CommandContext(ctx, "go", append([]string{"fix"}, args...)...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd.Run()
}
