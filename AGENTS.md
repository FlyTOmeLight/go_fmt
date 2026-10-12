# Repository Guidelines

## Project Structure & Module Organization

This module, `github.com/fsgo/go_fmt`, provides Go formatting and code rewrites. `main.go` builds the root `go_fmt` executable. `cmd/gorgeous/` contains the primary CLI; `cmd/goformat/` provides editor-oriented formatting. `gofmtapi/` exposes the public API.

Implementation packages live under `internal/`: `gofmt` coordinates formatting, `simplify` applies rewrites, `ximports` groups imports, and `xpasser` loads Go packages. Tests sit beside source files. Package-local `testdata/` directories contain fixtures, including nested Go modules. GitHub workflows live in `.github/workflows/`.

## Build, Test, and Development Commands

Use Go 1.26 or newer, as declared in `go.mod`. Run commands from the repository root:

- `go build -v`: build the root executable.
- `go build ./...`: compile all packages and CLI entry points.
- `go run ./cmd/gorgeous -d ./...`: inspect formatting differences; exit nonzero when changes are needed.
- `go run ./cmd/gorgeous path/to/file.go`: format a specific file in place.
- `go test -v ./...`: run the full test suite.
- `go vet ./...`: check common Go mistakes.
- `staticcheck ./...`: run the static analysis used by CI; install with `go install honnef.co/go/tools/cmd/staticcheck@latest`.

CI also runs `./go_fmt -rr -d ./...` after building.

## Coding Style & Naming Conventions

Use `gofmt` formatting and tabs for Go indentation. Keep imports grouped as standard library, third-party dependencies, then this module. Use short lowercase package names, exported `MixedCaps` identifiers, and descriptive unexported names. Keep formatter behavior in its responsible internal package; update public API documentation when its contract changes.

## Testing Guidelines

Tests use Go's `testing` package, with `fst` assertions and `go-cmp` comparisons. Use `*_test.go` files and `TestXxx` functions. Add regression fixtures for formatting changes: pair `*.go.input` with `*.go.want`, or follow the existing `input/` and `want/` layout. Inspect generated `.got` differences before changing expectations. Run focused tests, such as `go test ./internal/simplify`, then the full suite. No numeric coverage threshold is configured.

## Commit & Pull Request Guidelines

Recent commits use concise action-led subjects, such as `fix go vet` and `bump deps`; follow that style. For PRs, describe the behavior changed, include before/after code for rewrite rules, link relevant issues, and report validation commands. Keep changes focused and include matching fixtures.
