package xtest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fsgo/go_fmt/internal/common"
)

func TestGoldenMismatchFails(t *testing.T) {
	if os.Getenv("GOFMT_GOLDEN_PROBE") == "1" {
		dir := t.TempDir()
		input := filepath.Join(dir, "sample.go.input")
		if err := os.WriteFile(input, []byte("package sample\nvar A = 1\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(
			filepath.Join(dir, "sample.go.want"),
			[]byte("intentionally wrong\n"),
			0o600,
		); err != nil {
			t.Fatal(err)
		}
		CheckFileAuto(t, input, func(*common.Request) {})
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=^TestGoldenMismatchFails$")
	cmd.Env = append(os.Environ(), "GOFMT_GOLDEN_PROBE=1")
	out, err := cmd.CombinedOutput()
	if err == nil || !strings.Contains(string(out), "formatted output differs") {
		t.Fatalf("golden mismatch must fail the test: %v\n%s", err, out)
	}
}
