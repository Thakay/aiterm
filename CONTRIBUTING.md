# Contributing to aiterm

Thanks for your interest in aiterm! Bug reports, ideas, documentation fixes and code
are all welcome, whether it is your first open source contribution or your hundredth.

This project follows the [Code of Conduct](CODE_OF_CONDUCT.md). By taking part, you
agree to uphold it. Please report security problems privately as described in
[SECURITY.md](SECURITY.md), not in a public issue.

## Ways to contribute

- **Report a bug** with the [bug report form](https://github.com/Thakay/aiterm/issues/new/choose).
- **Suggest a feature** with the feature request form. The [roadmap](README.md#roadmap)
  lists ideas that are already planned.
- **Improve the docs**: README fixes, examples for other OpenAI compatible endpoints,
  typos.
- **Write code**: issues labeled
  [`good first issue`](https://github.com/Thakay/aiterm/labels/good%20first%20issue) and
  [`help wanted`](https://github.com/Thakay/aiterm/labels/help%20wanted) are a good
  place to start.

For anything bigger than a small fix (a new flag, a new provider, a change in
behavior), please open an issue first so we can agree on the approach before you
spend time on it.

## Prerequisites

- **Go 1.26 or newer.** CI tests with the two Go releases that still get security
  fixes. Go 1.21 and later download the required toolchain automatically.
- **Linux or macOS.** aiterm runs commands through `sh -c`, so Windows is not
  supported yet. On Windows, develop inside WSL.
- **Git and make.**
- **A C compiler** (gcc or clang) on Linux for the race detector, which `make test`
  uses. On macOS, the Xcode Command Line Tools (`xcode-select --install`) provide git
  and make.
- **Optional: [golangci-lint](https://golangci-lint.run) v2** for `make lint` and
  `make fmt`. CI uses v2.14. Without it, `gofmt -w .` formats the code.
- **Optional: [GoReleaser](https://goreleaser.com) v2** for `make snapshot`, to try the
  release build locally.

## Getting started

1. Fork the repository on GitHub.
2. Clone your fork and add the main repository as `upstream`:

   ```bash
   git clone https://github.com/<your-username>/aiterm.git
   cd aiterm
   git remote add upstream https://github.com/Thakay/aiterm.git
   ```

   If you only want to build and look around, `git clone https://github.com/Thakay/aiterm.git`
   is enough.

3. Create a branch for your change:

   ```bash
   git checkout -b fix/short-description
   ```

4. Build and run it:

   ```bash
   make build
   ./aiterm -version
   ./aiterm "list all go files in this directory"
   ```

   Running a real request needs an API key in `OPENAI_KEY`. To avoid API costs while
   developing, you can use a local model with Ollama (see
   [Local models with Ollama](README.md#local-models-with-ollama)). The tests never
   need a key and never call a real API.

Keep your branch up to date with `git fetch upstream && git rebase upstream/main`.

## Running the checks

The [Makefile](Makefile) wraps the commands you need:

| Command         | What it does                                                    |
|-----------------|-----------------------------------------------------------------|
| `make build`    | Builds the `aiterm` binary in the repository root               |
| `make test`     | Runs the tests with the race detector (`go test -race ./...`)   |
| `make cover`    | Runs the tests with coverage reporting                          |
| `make fuzz`     | Fuzzes the model reply validation for 30 seconds                |
| `make lint`     | Runs `golangci-lint run`                                        |
| `make fmt`      | Formats the code with `gofmt` and `goimports` (needs golangci-lint) |
| `make tidy`     | Runs `go mod tidy`                                              |
| `make vuln`     | Checks for known vulnerabilities with govulncheck               |
| `make snapshot` | Builds the release archives into `dist/` with GoReleaser, without publishing |

If you prefer the raw commands:

```bash
go build -o aiterm .                      # build
go test -race ./...                       # test
go test -race -run TestValidateCmd ./...  # run one test
go vet ./...                              # static checks
gofmt -l .                                # list files that need formatting (should print nothing)
golangci-lint run                         # lint
go mod tidy                               # tidy go.mod and go.sum
goreleaser release --snapshot --clean     # local release build into dist/
```

Before you open a pull request, make sure `make test` and `make lint` pass and `gofmt`
reports nothing. If you changed dependencies, run `make tidy` too.

### Continuous integration

Every pull request and every push to `main` runs these checks in GitHub Actions:

- Build, `go vet` and the tests with the race detector, on Linux and macOS, with the
  two supported Go releases.
- 30 seconds of fuzzing of the model reply validation.
- golangci-lint, which also checks formatting with `gofmt` and `goimports`, and a check
  that `go.mod` and `go.sum` are tidy.
- `goreleaser check` and a GoReleaser snapshot build, which make sure every release
  archive the README links to is produced.
- CodeQL code scanning and govulncheck (both also run weekly), and OpenSSF Scorecard
  on `main`.

A pull request can be merged once all checks are green and a maintainer has approved it.

## Project layout

aiterm is a single `main` package:

| File                    | Purpose                                                                 |
|-------------------------|-------------------------------------------------------------------------|
| `main.go`               | Entry point: flag parsing, environment variables, version output, exit codes |
| `app.go`                | The interactive menu loop: shows the command, runs it, copies it, edits it, sends follow-up requests |
| `openAI.go`             | Client for the OpenAI chat completions API (and compatible endpoints)  |
| `providers.go`          | The `APIProvider` interface that every model provider implements       |
| `errors.go`             | Typed errors, such as `APIKeyError` and `OAIAPIError`                  |
| `*_test.go`             | Tests for the file of the same name (`main_test.go` also covers end to end runs and exit codes) |
| `testhelpers_test.go`   | Shared test helpers: `fakeProvider`, a scripted `APIProvider`, and `chatServer`, an `httptest` server that mimics the chat completions endpoint |

The tests drive `app.go` with `fakeProvider` and test the OpenAI client against
`chatServer`, so they never call a real API and never need a key. Please keep it that
way: new tests must not touch the network.

### Guidelines for code changes

- Never run a command without the user pressing `y`. This is the core safety promise
  of aiterm. What the terminal shows must be exactly what runs, which is why replies
  with hidden characters are refused.
- Never print, log or store the API key, and only send it to the configured endpoint.
- Keep the exit codes stable: `0` for success, help and `-version`, `2` for usage
  errors, `1` for runtime errors.
- Flags take precedence over environment variables, which take precedence over the
  defaults. New settings should follow the same pattern.
- Wrap errors with `%w` or one of the types in `errors.go` so callers can use
  `errors.Is` and `errors.As`.
- Prefer the standard library. Discuss new dependencies in an issue first.
- Update [README.md](README.md) when you change flags, environment variables or
  behavior users can see.

## Adding a new provider

aiterm talks to models through the `APIProvider` interface in `providers.go`:

```go
type APIProvider interface {
	fetch(userRequest string, opts ...FetchConfig) (string, error)
	hasAPIKey() bool
	handleAPIError(err error) error
	newFetchConfig(withCtxt bool) FetchConfig
	setAPIKey(apikey string)
}
```

Many services already offer an OpenAI compatible endpoint and work today with `-url`.
A new provider is only needed for an API with a different request format. To add one:

1. Open an issue to agree on the flags or environment variables that select it.
2. Create a file such as `anthropic.go` with a type that implements `APIProvider`.
   The methods are unexported, so the type lives in package `main`.
   - `fetch` sends the request and returns the model's raw reply. The app strips code
     fences and detects the `not a command` reply, so you do not need to.
   - `newFetchConfig(true)` keeps the conversation history for the next `fetch`, and
     `newFetchConfig(false)` starts a new conversation. Only add a request and its
     reply to the history after the request succeeded.
   - `handleAPIError` should return an `*APIKeyError` when the key is missing or
     rejected, so the app can tell the user how to fix it.
   - Return the typed errors from `errors.go` for marshaling, request and response
     failures.
3. Construct your provider in `run` in `main.go` based on the new configuration.
4. Add tests in a matching `_test.go` file against an `httptest` server, like
   `openAI_test.go` does.
5. Document the provider in README.md and add a CHANGELOG.md entry.

## Commit messages

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/):

```text
<type>(<optional scope>): <short summary in the imperative mood>

<optional body: what changed and why>

Closes #123
```

Use one of these types:

| Type       | Use it for                                          |
|------------|-----------------------------------------------------|
| `feat`     | A new feature                                       |
| `fix`      | A bug fix                                           |
| `docs`     | Documentation only                                  |
| `test`     | Adding or fixing tests                              |
| `ci`       | GitHub Actions workflows                            |
| `build`    | The Makefile, GoReleaser, Go version or dependencies |
| `refactor` | Code changes that neither fix a bug nor add a feature |
| `chore`    | Maintenance, such as release preparation            |

Examples:

```text
feat: add -explain flag to describe a command before running it
fix(openai): report the HTTP status for non JSON error pages
docs: add an LM Studio example to the README
```

Mark breaking changes with `!` after the type (`feat!: ...`) and explain them in a
`BREAKING CHANGE:` footer. Reference the issue a commit resolves with `Closes #N` (or
`Fixes #N`) so GitHub closes it on merge. Please use the same format for the pull
request title.

## Pull requests

Before you open a pull request, check that:

- [ ] The change is focused on one thing. Unrelated fixes go in separate pull requests.
- [ ] Tests are added or updated for new behavior and bug fixes.
- [ ] `make test` and `make lint` pass, and `gofmt -l .` prints nothing.
- [ ] User facing changes have an entry under `## [Unreleased]` in
      [CHANGELOG.md](CHANGELOG.md), in the right subsection (`Added`, `Changed`,
      `Deprecated`, `Removed`, `Fixed` or `Security`).
- [ ] README.md and the `aiterm -h` help text are updated if flags or behavior changed.
- [ ] Commits and the pull request title follow Conventional Commits.
- [ ] The description links the issue it resolves (`Closes #N`).

The pull request template includes this checklist. The maintainer reviews pull
requests as time allows. If you have not heard back within a week, feel free to leave
a comment on the pull request.

## Releasing (maintainers)

aiterm uses [Semantic Versioning](https://semver.org/spec/v2.0.0.html). While the
version is below 1.0.0, new features and breaking changes bump the minor version and
bug fixes bump the patch version.

1. Make sure `main` is green in CI. Optionally run `make snapshot` to check the release
   build locally.
2. In [CHANGELOG.md](CHANGELOG.md), move the entries under `## [Unreleased]` into a new
   `## [X.Y.Z] - YYYY-MM-DD` section and leave an empty `## [Unreleased]` heading above
   it. Update the compare links at the bottom of the file:

   ```markdown
   [Unreleased]: https://github.com/Thakay/aiterm/compare/vX.Y.Z...HEAD
   [X.Y.Z]: https://github.com/Thakay/aiterm/compare/vPREVIOUS...vX.Y.Z
   ```

3. Open a pull request titled `chore(release): vX.Y.Z` and merge it.
4. Tag the merge commit on `main` and push the tag, from a clone whose `origin` is
   `https://github.com/Thakay/aiterm.git`:

   ```bash
   git checkout main && git pull origin main
   git tag -a vX.Y.Z -m "aiterm vX.Y.Z" && git push origin vX.Y.Z
   ```

5. The Release workflow runs GoReleaser. It builds the Linux and macOS binaries for
   amd64 and arm64, packages them with a `checksums.txt` file, and publishes the GitHub
   release with the `X.Y.Z` section of CHANGELOG.md as the release notes.
6. Check the release page, then confirm that
   `go install github.com/Thakay/aiterm@vX.Y.Z` works and that `aiterm -version`
   reports the new version.

## Getting help

If something in this guide is unclear or out of date, or you are stuck, open an
[issue](https://github.com/Thakay/aiterm/issues/new/choose) and ask. Questions are
welcome.

## License

aiterm is released under the [MIT License](LICENSE). By contributing, you agree that
your contributions are licensed under the same license.
