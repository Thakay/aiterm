# AGENTS.md

Guidance for coding agents (Codex, Claude Code and others) working on aiterm. Human
contributors should read [CONTRIBUTING.md](CONTRIBUTING.md), which has the full details.

## Project

aiterm is a Go CLI that turns a natural language request into a shell command through
an OpenAI compatible chat completions API, shows it, and runs it only after the user
enters `y`. It is a single `main` package:

- `main.go`: flag and environment parsing, exit codes, version output
- `app.go`: the interactive menu, reply cleaning and validation, command execution
- `openAI.go`: the chat completions client
- `providers.go`: the `APIProvider` interface
- `errors.go`: typed errors that support `errors.Is` and `errors.As`

## Commands

Run these before you finish a change. All of them must pass.

```bash
make test    # go test -race ./...
make lint    # golangci-lint run (v2 config in .golangci.yml)
go mod tidy  # go.mod and go.sum must not change
```

Also useful: `make fuzz` (fuzzes `ValidateCmd`), `make vuln` (govulncheck) and
`make snapshot` (GoReleaser release build into `dist/`, nothing is published).

## Rules

- Never run a command unless the user entered `y`. What the terminal shows must be
  exactly what runs: keep refusing replies with control or invisible characters
  (`hasHiddenRunes` in `app.go`).
- Tests must never touch the network or need an API key. Use `fakeProvider` and
  `chatServer` from `testhelpers_test.go`. Add a test for every behavior change.
- Never print, log or store the API key, and only send it to the configured endpoint.
- Settings resolve as flag, then environment variable, then default. Flags must come
  before the request.
- Keep exit codes stable: `0` success, `2` usage error, `1` runtime error.
- Prefer the standard library; discuss new dependencies in an issue first.
- Update `README.md` for user visible changes and add an entry under
  `## [Unreleased]` in `CHANGELOG.md`.
- Commit messages and pull request titles follow Conventional Commits
  (`feat:`, `fix:`, `docs:`, `test:`, `ci:`, `build:`, `refactor:`, `chore:`).
