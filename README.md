# aiterm

[![CI](https://github.com/Thakay/aiterm/actions/workflows/ci.yml/badge.svg)](https://github.com/Thakay/aiterm/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/Thakay/aiterm?sort=semver)](https://github.com/Thakay/aiterm/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/Thakay/aiterm)](https://goreportcard.com/report/github.com/Thakay/aiterm)
[![Go Version](https://img.shields.io/github/go-mod/go-version/Thakay/aiterm)](go.mod)
[![MIT License](https://img.shields.io/badge/license-MIT-green.svg)](LICENSE)

`aiterm` translates natural language into shell commands, right in your terminal.
Describe what you want, review the command it suggests, then run it, copy it, edit it,
or refine it with a follow-up request. No more searching for that `find` or `tar`
incantation.

It is written in Go, ships as a single binary, and works with OpenAI or any
OpenAI compatible API, including local models served by Ollama or LM Studio.

<p align="center">
  <img src="usage1.gif" alt="aiterm turning a natural language request into a shell command" />
</p>

## Features

- **Natural language to commands**: describe the task, get a single command tailored
  to your OS (macOS BSD tools or Linux GNU tools).
- **You stay in control**: nothing runs until you press `y`. You can also copy the
  command, edit it before running, or exit.
- **Follow-up requests**: refine the last answer with context ("now include hidden
  files") or start fresh without it.
- **Bring your own model**: pick any model with `-model`, and point `-url` at any
  OpenAI compatible endpoint (Ollama, LM Studio, OpenRouter, ...).
- **Single static binary** for Linux and macOS on amd64 and arm64.

## Installation

### Prebuilt binaries

Download the archive for your platform from the
[latest release](https://github.com/Thakay/aiterm/releases/latest), extract it and put
`aiterm` on your `PATH`. For example:

```bash
# macOS on Apple silicon (use Darwin_x86_64 for Intel Macs)
curl -sSL https://github.com/Thakay/aiterm/releases/latest/download/aiterm_Darwin_arm64.tar.gz | tar -xz aiterm
sudo mv aiterm /usr/local/bin/

# Linux on x86_64 (use Linux_arm64 for ARM)
curl -sSL https://github.com/Thakay/aiterm/releases/latest/download/aiterm_Linux_x86_64.tar.gz | tar -xz aiterm
sudo mv aiterm /usr/local/bin/
```

Each release includes a `checksums.txt` file to verify the download.

### With Go

Requires Go 1.22 or newer.

```bash
go install github.com/Thakay/aiterm@latest
```

### From source

```bash
git clone https://github.com/Thakay/aiterm.git
cd aiterm
go build -o aiterm .
```

## Configuration

`aiterm` needs an API key. Flags take precedence over environment variables.

| Flag       | Environment variable | Default                                      | Description                                          |
|------------|----------------------|----------------------------------------------|------------------------------------------------------|
| `-key`     | `OPENAI_KEY`         |                                              | API key sent as a bearer token                       |
| `-model`   | `AITERM_MODEL`       | `gpt-4.1-mini`                               | Model used to generate commands                      |
| `-url`     | `AITERM_URL`         | `https://api.openai.com/v1/chat/completions` | Chat completions endpoint of an OpenAI compatible API |
| `-version` |                      |                                              | Print the version and exit                           |

Prefer the environment variable over `-key` so the key does not end up in your shell
history. If no key is set, `aiterm` asks for one and uses it for the current session.

```bash
export OPENAI_KEY="sk-..."
```

### Local models with Ollama

```bash
export AITERM_URL="http://localhost:11434/v1/chat/completions"
export AITERM_MODEL="llama3.2"
export OPENAI_KEY="ollama"   # Ollama ignores the key, but aiterm expects one to be set
```

## Usage

Pass your request as arguments. Quotes are optional.

```bash
aiterm "find all the files that contain the word foo in the parent directory"
aiterm show the 10 largest files in this folder
```

`aiterm` shows the suggested command and a menu:

| Key | Action                                                              |
|-----|---------------------------------------------------------------------|
| `y` | Run the command (its output streams straight to your terminal)      |
| `c` | Copy the command to the clipboard and exit                          |
| `g` | Copy the command, then paste an edited version to run with `aiterm` |
| `r` | Send a follow-up request that keeps the conversation context        |
| `w` | Send a new request without the previous context                     |
| `q` | Quit                                                                |

If a command fails, `aiterm` prints the exit status and brings the menu back so you
can edit it or ask for another one.

On Linux, copying to the clipboard needs `xclip`, `xsel` or `wl-clipboard`. Without
one of them, `aiterm` prints the command so you can copy it yourself.

> [!WARNING]
> Commands come from a language model and can be wrong or destructive. Always read a
> command before you run it.

## Development

```bash
go test -race ./...      # run the tests
go vet ./...             # static checks
golangci-lint run        # lint (https://golangci-lint.run)
```

Every pull request runs the same checks in CI on Linux and macOS. Releases are built
by [GoReleaser](https://goreleaser.com) when a `v*` tag is pushed. See
[CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Roadmap

- Shell completions and a man page
- A `-explain` mode that describes what a command does before you run it
- Native support for more providers (Anthropic, Gemini)
- Windows support (PowerShell commands)
- A Homebrew tap

Ideas and feedback are welcome in the [issue tracker](https://github.com/Thakay/aiterm/issues).

## Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) and our
[Code of Conduct](CODE_OF_CONDUCT.md) before opening a pull request. Report security
issues privately as described in [SECURITY.md](SECURITY.md).

## License

`aiterm` is licensed under the [MIT License](LICENSE).

## Contact

For questions and support, [open an issue](https://github.com/Thakay/aiterm/issues/new/choose).
