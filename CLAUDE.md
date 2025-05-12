# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build Commands
- Build: `go build`
- Run: `./iecs`
- Dev environment: `devenv up`
- Update dependencies: `gomod2nix`

## Test Commands
- Run tests: `go test ./...`
- Test with coverage: `go test -coverprofile=coverage.out ./...`
- View coverage: `go tool cover -html=coverage.out`

## Code Style Guidelines
- Formatting: Use `gofmt` (enforced via git hook)
- Imports: Standard Go style (stdlib first, then external)
- Error handling: Use `fmt.Errorf` with context and error wrapping `%w`
- Naming conventions: Follow Go standards (CamelCase for exported items)
- Package structure:
  - `client`: AWS client implementation
  - `cmd`: CLI commands
  - `selector`: Selection logic components
- Interface-based dependency injection
- Descriptive error messages with resource context

## Git Hooks
Git hooks are configured to automatically run:
- `gofmt`: Code formatting
- `golangci-lint`: Static code analysis
- `gomod2nix`: Dependency tracking
- `nixpkgs-fmt`: Nix formatting