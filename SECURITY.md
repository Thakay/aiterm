# Security Policy

## Supported versions

Security fixes are released for the latest minor version only. Please upgrade to the
newest release before reporting a problem.

| Version | Supported |
|---------|-----------|
| 0.2.x   | Yes       |
| < 0.2   | No        |

Run `aiterm -version` to see which version you have.

## Reporting a vulnerability

Please report security problems privately through GitHub private vulnerability
reporting:

**<https://github.com/Thakay/aiterm/security/advisories/new>**

Do not open a public issue, pull request or discussion for a security problem, and
do not share details publicly until a fix is released.

Please include:

- The aiterm version (`aiterm -version`), your OS and architecture, and how you
  installed it (release binary, `go install`, or built from source).
- The endpoint type you use (OpenAI, Ollama, another OpenAI compatible API) and the
  model, if they matter for the problem.
- A description of the problem and its impact: what an attacker can do, and what they
  need to control first (for example the endpoint, a model reply, or the local machine).
- Step by step instructions or a proof of concept that reproduces it. For problems
  triggered by a model reply, the exact reply text is the most useful thing you can send.
- A suggested fix, if you have one.

Never include a real API key in a report. Replace keys, tokens and private hostnames
with placeholders.

## What to expect

aiterm is maintained by volunteers, so responses are best effort. The maintainer will:

- Acknowledge your report within 7 days.
- Confirm whether it is a vulnerability, and keep you updated while a fix is prepared.
- Release a fix, publish a GitHub security advisory (requesting a CVE when it is
  warranted), and note the fix in the `Security` section of [CHANGELOG.md](CHANGELOG.md).
- Credit you in the advisory, unless you prefer to stay anonymous.

If the form is not available, or you have not received a reply within 7 days, open a
public issue that only asks the maintainer, [@Thakay](https://github.com/Thakay), to
get in touch about a security report. Do not include any details in that issue.

## Security model

Knowing how aiterm handles commands and keys helps decide whether something is a
vulnerability.

- **Commands only run after you confirm them.** aiterm shows every command the model
  suggests and runs it with `sh -c` only after you press `y`. This also applies to a
  command you edited with `g`. The command runs with your user's permissions, in
  your shell environment.
- **What you see is what runs.** Replies that contain control characters, terminal
  escape sequences, bidi overrides or zero width characters are refused, and
  multi line commands are flagged before you confirm them.
- **What is sent to the endpoint.** Each request contains a system prompt that names
  your operating system, the requests you typed, and the model's earlier replies in
  the same session (unless you start over with `w`). aiterm does not send files,
  environment variables or command output.
- **API keys.** The key is read from the `OPENAI_KEY` environment variable or the
  `-key` flag, or typed at a prompt (without echo) when neither is set or the key is
  rejected. It is sent only to the configured endpoint, as a bearer token in the
  `Authorization` header. aiterm does not write it to disk, and `aiterm -h` does not
  print it. A key passed with `-key` can
  end up in your shell history and is visible to other local users in the process
  list, so prefer the environment variable.
- **Endpoints.** `-url` and `AITERM_URL` can point at any endpoint, and aiterm sends
  your key and requests there. The endpoint decides which commands you are offered.
  Only use endpoints you trust, and use `https://` for anything that is not on your
  own machine (aiterm warns about plain `http://` to another host).

### In scope

For example:

- A way to make aiterm run a command without the user pressing `y`.
- A command that runs differently from the one shown, for example because hidden
  characters or terminal escape sequences in a model reply change what is displayed.
- The API key being sent anywhere other than the configured endpoint, or being
  written to disk, logs or terminal output.
- Problems in the release process or release artifacts, such as the GitHub Actions
  workflows, the published binaries or `checksums.txt`.

### Out of scope

- The model suggesting a wrong, harmful or destructive command that the user then
  chose to run. Always read a command before you run it.
- An endpoint you configured returning a malicious command, as long as aiterm still
  shows it and waits for confirmation.
- Bugs in the model or in third party services (OpenAI, Ollama, LM Studio and others).
  Please report those to the provider.
- The `-key` value being visible in shell history or the process list, as described
  above.
- Attacks that require control of your user account, shell or environment variables.
- Vulnerabilities in dependencies that do not affect aiterm. Dependabot keeps
  dependencies up to date; please report those upstream.
- Windows, which is not supported.
