# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.2.0] - 2026-10-04

### Added

- `-model` flag and `AITERM_MODEL` environment variable to choose the model.
- `AITERM_URL` environment variable to point `aiterm` at any OpenAI compatible
  chat completions endpoint (Ollama, LM Studio, OpenRouter, Azure OpenAI proxies, ...),
  alongside the existing `-url` flag.
- `-version` flag. Release builds report the version, commit and build date.
- Prompts no longer need quotes: `aiterm list all go files` works.
- The system prompt now names the operating system, so macOS users get BSD
  flavoured commands and Linux users get GNU flavoured ones.
- Model replies wrapped in markdown code fences, inline backticks or a `$ `
  prompt are cleaned up before they are shown or run.
- Prebuilt binaries for Linux and macOS (amd64 and arm64) on every release, with checksums.
- `go install github.com/Thakay/aiterm@latest` support.
- Unit and end-to-end tests for `main.go`, `app.go`, the OpenAI client and the
  error types ([#1](https://github.com/Thakay/aiterm/issues/1),
  [#2](https://github.com/Thakay/aiterm/issues/2)).
- Continuous integration on Linux and macOS (tests with the race detector,
  `go vet`, `gofmt`, `golangci-lint`), Dependabot updates and automated releases.
- Contributing guide, security policy, code of conduct, and issue and pull request templates.

### Changed

- The default model is now `gpt-4.1-mini`. The previous default, `gpt-3.5-turbo`,
  is scheduled to be shut down by OpenAI on 2026-10-23.
- Commands now stream their output straight to the terminal, so long running and
  interactive commands behave as if you typed them.
- When a command fails, `aiterm` reports the exit status and returns to the menu
  so you can edit it or ask for another one, instead of exiting.
- When the clipboard is unavailable (for example a headless Linux box without
  `xclip`/`xsel`), the command is printed instead of aborting.
- Running `aiterm` without a prompt or with an unknown flag now exits with status 2.
- The Go module path is now `github.com/Thakay/aiterm` (was `github.com/thakay/goterm`).

### Fixed

- A missing API key was never detected, so `aiterm` sent unauthenticated requests
  instead of asking for a key.
- `aiterm -h` printed the value of `OPENAI_KEY` as the flag default.
- HTTP requests had no timeout and could hang forever.
- A non JSON error page (for example from a proxy) was reported as an unmarshaling
  error instead of the HTTP status.
- `aiterm` looped forever when standard input was closed after a "not a command" reply.
- A failed request left the user message in the conversation history.
- Error types now implement `Unwrap`, and the unmarshaling error no longer claims
  to be a marshaling error.
- The README clone URL pointed to a placeholder account.

## [0.1.0-beta.1] - 2024-02-05

### Added

- First beta: translate a natural language request into a Unix command with the
  OpenAI API, then run it, copy it to the clipboard, edit it, or send a follow-up
  request with or without the previous context.

[Unreleased]: https://github.com/Thakay/aiterm/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/Thakay/aiterm/compare/v0.1.0-beta.1...v0.2.0
[0.1.0-beta.1]: https://github.com/Thakay/aiterm/releases/tag/v0.1.0-beta.1
