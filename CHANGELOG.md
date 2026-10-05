# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- `-print` flag that writes the suggested command to stdout and exits, for
  scripts and shell keybindings. It never prompts or runs anything.
- Ship a man page for aiterm in release archives.
- Add next-step hints for common API errors and refused local-model connections.
- Shell completions for bash, zsh and fish in `completions/`, shipped in
  the release archives. See the README's Shell completions section for
  how to enable them.

## [0.2.0] - 2026-10-04

### Added

- Follow-up requests from the menu: `r` sends a new request that keeps the
  conversation context, `w` starts a new conversation without it.
- `-model` flag and `AITERM_MODEL` environment variable to choose the model. Any
  model works, including reasoning models.
- `AITERM_URL` environment variable to point `aiterm` at any OpenAI compatible
  chat completions endpoint (Ollama, LM Studio, OpenRouter, ...), alongside the
  existing `-url` flag.
- `-timeout` flag and `AITERM_TIMEOUT` environment variable (default `2m`), so slow
  local models can be given more time.
- `-version` flag. Release builds report the version, commit and commit date.
- Prompts no longer need quotes: `aiterm list all go files` works.
- The system prompt now names the operating system, so macOS users get BSD
  flavoured commands and Linux users get GNU flavoured ones.
- Model replies wrapped in markdown code fences, inline backticks or `$ ` prompts
  are cleaned up before they are shown or run.
- A warning before running a command that spans several lines.
- Ctrl+C stops a running command and returns to the menu instead of quitting `aiterm`.
- Prebuilt binaries for Linux and macOS (amd64 and arm64) on every release, with
  checksums and reproducible builds.
- `go install github.com/Thakay/aiterm@latest` support.
- Unit, end-to-end and fuzz tests for `main.go`, `app.go`, the OpenAI client and the
  error types ([#1](https://github.com/Thakay/aiterm/issues/1),
  [#2](https://github.com/Thakay/aiterm/issues/2)).
- Continuous integration on Linux and macOS (race-enabled tests, fuzzing, `go vet`,
  golangci-lint, a GoReleaser snapshot build), CodeQL, govulncheck, OpenSSF
  Scorecard, Dependabot updates and automated releases.
- A Makefile, a contributing guide, a security policy, a code of conduct, issue and
  pull request templates, and an `AGENTS.md` for coding agents.

### Changed

- The default model is now `gpt-4.1-mini`. The previous default, `gpt-3.5-turbo`,
  is scheduled to be shut down by OpenAI on 2026-10-23.
- Requests only send the model and the messages. `max_tokens`, `temperature` and the
  other sampling parameters are left to the server defaults, because reasoning models
  reject them.
- Commands now stream their output straight to the terminal, so long running and
  interactive commands behave as if you typed them.
- When a command fails, `aiterm` reports the exit status and returns to the menu
  so you can edit it or ask for another one, instead of exiting.
- When the clipboard is unavailable (for example a headless Linux box without
  `xclip`/`xsel`), the command is printed instead of aborting.
- Running `aiterm` without a prompt now exits with status 2 (it used to exit with 0).
- Building from source now requires Go 1.26, the oldest Go release that still gets
  security fixes. Go 1.21 and later download it automatically.
- The Go module path is now `github.com/Thakay/aiterm` (was `github.com/thakay/goterm`).

### Fixed

- A missing API key was never detected, so `aiterm` sent unauthenticated requests
  instead of asking for a key.
- HTTP requests had no timeout and could hang forever.
- A non JSON error page (for example from a proxy) was reported as an unmarshaling
  error instead of the HTTP status.
- `aiterm` looped forever when standard input was closed after a "not a command" reply.
- A failed request left the user message in the conversation history.
- Error types now implement `Unwrap`, and the unmarshaling error no longer claims
  to be a marshaling error.
- The README clone URL pointed to a placeholder account.

### Security

- Replies that contain control characters, terminal escape sequences, bidi overrides
  or zero width characters are refused. They could make the terminal show a different
  command from the one `sh` would run.
- A flag typed after the request (such as `aiterm list files -key sk-...`) is now an
  error. Before, it was sent to the model as part of the request.
- `aiterm -h` printed the value of `OPENAI_KEY` as the default of `-key`.
- An API key typed at the prompt is no longer echoed to the terminal.
- Responses larger than 1 MiB are rejected instead of being read into memory.
- `aiterm` warns when the API key would be sent over plain http to another machine.

## [0.1.0-beta.1] - 2024-02-05

### Added

- First beta: translate a natural language request into a Unix command with the
  OpenAI API, then run it, copy it to the clipboard, or edit it.

[Unreleased]: https://github.com/Thakay/aiterm/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/Thakay/aiterm/compare/v0.1.0-beta.1...v0.2.0
[0.1.0-beta.1]: https://github.com/Thakay/aiterm/releases/tag/v0.1.0-beta.1
