package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func skipWithoutShell(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("executing commands requires a POSIX sh")
	}
}

func TestNewAppDefaults(t *testing.T) {
	p := &fakeProvider{apiKey: "k"}
	app := NewApp(p, "list files")

	if app.Client != p {
		t.Errorf("Client = %v, want the provider passed in", app.Client)
	}
	if app.userRequest != "list files" {
		t.Errorf("userRequest = %q, want %q", app.userRequest, "list files")
	}
	if app.in != os.Stdin || app.out != os.Stdout || app.errOut != os.Stderr {
		t.Error("NewApp should be wired to the process standard streams")
	}
	if app.reader == nil || app.copyToClipboard == nil || app.fetchCfg == nil {
		t.Error("NewApp left a dependency nil")
	}
}

func TestValidateCmd(t *testing.T) {
	tests := []struct {
		name   string
		in     string
		wantOK bool
		want   string
	}{
		{"plain command", "ls -la", true, "ls -la"},
		{"surrounding whitespace", "  \n ls -la \n", true, "ls -la"},
		{"not a command", "not a command", false, ""},
		{"not a command any case with period", "Not a command.", false, ""},
		{"empty", "   ", false, ""},
		{"fenced with language", "```bash\nfind . -name '*.go'\n```", true, "find . -name '*.go'"},
		{"fenced with sh language", "```sh\nls\n```", true, "ls"},
		{"fenced without language", "```\ndu -sh *\n```", true, "du -sh *"},
		{"fenced single line", "```ls -la```", true, "ls -la"},
		{"fenced multi line", "```\ncd /tmp\nls\n```", true, "cd /tmp\nls"},
		{"inline backticks", "`grep -rn foo .`", true, "grep -rn foo ."},
		{"command substitution is preserved", "echo `date`", true, "echo `date`"},
		{"shell prompt prefix", "$ df -h", true, "df -h"},
		{"fenced not a command", "```\nnot a command\n```", false, ""},
		{"fenced with any info string", "```console-session\nls\n```", true, "ls"},
		{"windows line endings", "```bash\r\nls -la\r\n```\r\n", true, "ls -la"},
		{"shell prompt on every line", "$ cd /tmp\n$ ls", true, "cd /tmp\nls"},
		{"tabs are allowed", "printf 'a\tb'", true, "printf 'a\tb'"},
		{"carriage return hides text", "touch pwned #\rls -la", false, ""},
		{"escape sequence", "touch pwned; echo hi\x1b[2K\rls", false, ""},
		{"backspace", "rm -rf ~/x\b\b\b\b\b\b\b\b\bls", false, ""},
		{"bidi override", "echo \u202eecho safe", false, ""},
		{"zero width space", "ls\u200b -la", false, ""},
		{"invalid utf-8", "ls \xff", false, ""},
	}

	app := &App{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ok, got := app.ValidateCmd(tt.in)
			if ok != tt.wantOK || got != tt.want {
				t.Errorf("ValidateCmd(%q) = (%v, %q), want (%v, %q)", tt.in, ok, got, tt.wantOK, tt.want)
			}
		})
	}
}

func TestHandleEmptyAPIKey(t *testing.T) {
	t.Run("key already set", func(t *testing.T) {
		p := &fakeProvider{apiKey: "sk-test"}
		app, out, _, _ := testApp(t, p, "", "")
		ok, err := app.HandleEmptyAPIKey()
		if !ok || err != nil {
			t.Fatalf("got (%v, %v), want (true, nil)", ok, err)
		}
		if out.Len() != 0 {
			t.Errorf("should not prompt when a key is set, printed %q", out.String())
		}
	})

	t.Run("key entered at the prompt", func(t *testing.T) {
		p := &fakeProvider{}
		app, out, _, _ := testApp(t, p, "", "sk-typed\n")
		ok, err := app.HandleEmptyAPIKey()
		if !ok || err != nil {
			t.Fatalf("got (%v, %v), want (true, nil)", ok, err)
		}
		if p.apiKey != "sk-typed" {
			t.Errorf("api key = %q, want %q", p.apiKey, "sk-typed")
		}
		if !strings.Contains(out.String(), "OPENAI_KEY") {
			t.Errorf("prompt should mention OPENAI_KEY, got %q", out.String())
		}
	})

	t.Run("empty input exits", func(t *testing.T) {
		p := &fakeProvider{}
		app, _, _, _ := testApp(t, p, "", "\n")
		ok, err := app.HandleEmptyAPIKey()
		if ok || err != nil {
			t.Fatalf("got (%v, %v), want (false, nil)", ok, err)
		}
	})

	t.Run("closed stdin", func(t *testing.T) {
		p := &fakeProvider{}
		app, _, _, _ := testApp(t, p, "", "")
		ok, err := app.HandleEmptyAPIKey()
		var inputErr *InputReadError
		if ok || !errors.As(err, &inputErr) {
			t.Fatalf("got (%v, %v), want (false, *InputReadError)", ok, err)
		}
	})
}

func TestHandleCmdQuit(t *testing.T) {
	app, out, _, _ := testApp(t, &fakeProvider{}, "", "q\n")
	end, err := app.HandleCmd("ls -la")
	if !end || err != nil {
		t.Fatalf("got (%v, %v), want (true, nil)", end, err)
	}
	if !strings.Contains(out.String(), "Here is the command --> ls -la <--") {
		t.Errorf("menu does not show the command: %q", out.String())
	}
}

func TestHandleCmdExecute(t *testing.T) {
	skipWithoutShell(t)
	app, out, _, _ := testApp(t, &fakeProvider{}, "", "y\n")
	end, err := app.HandleCmd("echo hello-from-aiterm")
	if !end || err != nil {
		t.Fatalf("got (%v, %v), want (true, nil)", end, err)
	}
	if !strings.Contains(out.String(), "hello-from-aiterm\n") {
		t.Errorf("command output missing: %q", out.String())
	}
}

func TestHandleCmdExecuteFailureReturnsToMenu(t *testing.T) {
	skipWithoutShell(t)
	app, out, errOut, _ := testApp(t, &fakeProvider{}, "", "y\nq\n")
	end, err := app.HandleCmd("echo oops >&2; exit 3")
	if !end || err != nil {
		t.Fatalf("got (%v, %v), want (true, nil)", end, err)
	}
	if !strings.Contains(out.String(), "The command failed") || !strings.Contains(out.String(), "exit status 3") {
		t.Errorf("failure was not reported: %q", out.String())
	}
	if got := strings.Count(out.String(), "Here is the command"); got != 2 {
		t.Errorf("menu shown %d times, want 2 (once more after the failure)", got)
	}
	if !strings.Contains(errOut.String(), "oops") {
		t.Errorf("stderr of the command should be forwarded, got %q", errOut.String())
	}
}

func TestHandleCmdWarnsAboutMultiLineCommands(t *testing.T) {
	app, out, _, _ := testApp(t, &fakeProvider{}, "", "q\n")
	if _, err := app.HandleCmd("cd /tmp\nls"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Note: this command has 2 lines") {
		t.Errorf("multi line command should be flagged before the prompt: %q", out.String())
	}

	app, out, _, _ = testApp(t, &fakeProvider{}, "", "q\n")
	if _, err := app.HandleCmd("ls"); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(out.String(), "Note:") {
		t.Errorf("single line command should not be flagged: %q", out.String())
	}
}

// TestOnlyYRunsTheCommand guards the core safety promise: nothing runs unless
// the user enters y.
func TestOnlyYRunsTheCommand(t *testing.T) {
	skipWithoutShell(t)
	for _, choice := range []string{"", "n", "yes", "Y ", "x", "c", "q", "r\nnew request", "w\nnew request"} {
		t.Run(fmt.Sprintf("%q", choice), func(t *testing.T) {
			marker := filepath.Join(t.TempDir(), "ran")
			app, out, _, _ := testApp(t, &fakeProvider{}, "", choice+"\nq\n")
			if _, err := app.HandleCmd("touch " + marker); err != nil {
				t.Fatal(err)
			}
			_, statErr := os.Stat(marker)
			ranCommand := statErr == nil
			if want := strings.TrimSpace(choice) == "Y"; ranCommand != want {
				t.Errorf("command ran = %v, want %v (output %q)", ranCommand, want, out.String())
			}
		})
	}
}

func TestHandleCmdCopy(t *testing.T) {
	app, out, _, copied := testApp(t, &fakeProvider{}, "", "c\n")
	end, err := app.HandleCmd("ls -la")
	if !end || err != nil {
		t.Fatalf("got (%v, %v), want (true, nil)", end, err)
	}
	if len(*copied) != 1 || (*copied)[0] != "ls -la" {
		t.Errorf("clipboard got %q, want [\"ls -la\"]", *copied)
	}
	if !strings.Contains(out.String(), "Command copied to clipboard") {
		t.Errorf("missing confirmation: %q", out.String())
	}
}

func TestHandleCmdCopyFailurePrintsCommand(t *testing.T) {
	app, out, _, _ := testApp(t, &fakeProvider{}, "", "c\n")
	app.copyToClipboard = func(string) error { return errors.New("no clipboard utility") }
	end, err := app.HandleCmd("ls -la")
	if !end || err != nil {
		t.Fatalf("got (%v, %v), want (true, nil)", end, err)
	}
	if !strings.Contains(out.String(), "no clipboard utility") || !strings.Contains(out.String(), "\nls -la\n") {
		t.Errorf("should explain the failure and print the command: %q", out.String())
	}
}

func TestHandleCmdEditThenRun(t *testing.T) {
	skipWithoutShell(t)
	app, out, _, copied := testApp(t, &fakeProvider{}, "", "g\necho edited-command\ny\n")
	end, err := app.HandleCmd("echo original")
	if !end || err != nil {
		t.Fatalf("got (%v, %v), want (true, nil)", end, err)
	}
	if len(*copied) != 1 || (*copied)[0] != "echo original" {
		t.Errorf("clipboard got %q, want the original command", *copied)
	}
	if !strings.Contains(out.String(), "Here is the command --> echo edited-command <--") {
		t.Errorf("edited command not shown: %q", out.String())
	}
	if !strings.Contains(out.String(), "edited-command\n") || strings.Contains(out.String(), "Running: echo original") {
		t.Errorf("the edited command should run instead of the original: %q", out.String())
	}
}

func TestHandleCmdEditEmptyKeepsCommand(t *testing.T) {
	app, out, _, _ := testApp(t, &fakeProvider{}, "", "g\n\nq\n")
	app.copyToClipboard = func(string) error { return errors.New("headless") }
	end, err := app.HandleCmd("ls")
	if !end || err != nil {
		t.Fatalf("got (%v, %v), want (true, nil)", end, err)
	}
	if got := strings.Count(out.String(), "Here is the command --> ls <--"); got != 2 {
		t.Errorf("original command should be kept after empty edit, shown %d times: %q", got, out.String())
	}
}

func TestHandleCmdNewRequest(t *testing.T) {
	tests := []struct {
		choice      string
		wantContext bool
	}{
		{"r", true},
		{"R", true},
		{"w", false},
	}
	for _, tt := range tests {
		t.Run(tt.choice, func(t *testing.T) {
			p := &fakeProvider{}
			app, _, _, _ := testApp(t, p, "first", tt.choice+"\nsecond request\n")
			end, err := app.HandleCmd("ls")
			if end || err != nil {
				t.Fatalf("got (%v, %v), want (false, nil)", end, err)
			}
			if app.userRequest != "second request" {
				t.Errorf("userRequest = %q, want %q", app.userRequest, "second request")
			}
			app.fetchCfg(p)
			if len(p.withContext) != 1 || p.withContext[0] != tt.wantContext {
				t.Errorf("fetch config set withContext = %v, want %v", p.withContext, tt.wantContext)
			}
		})
	}
}

func TestHandleCmdUnknownChoice(t *testing.T) {
	app, out, _, _ := testApp(t, &fakeProvider{}, "", "x\nq\n")
	end, err := app.HandleCmd("ls")
	if !end || err != nil {
		t.Fatalf("got (%v, %v), want (true, nil)", end, err)
	}
	if !strings.Contains(out.String(), "unknown command") {
		t.Errorf("unknown choice not reported: %q", out.String())
	}
}

func TestHandleCmdClosedInput(t *testing.T) {
	app, _, _, _ := testApp(t, &fakeProvider{}, "", "")
	end, err := app.HandleCmd("ls")
	var inputErr *InputReadError
	if !end || !errors.As(err, &inputErr) {
		t.Fatalf("got (%v, %v), want (true, *InputReadError)", end, err)
	}
	if !errors.Is(err, io.EOF) {
		t.Errorf("error should wrap io.EOF, got %v", err)
	}
}

func TestReadInputWithoutTrailingNewline(t *testing.T) {
	app, _, _, _ := testApp(t, &fakeProvider{}, "", "  y  ")
	got, err := app.readInput()
	if err != nil || got != "y" {
		t.Fatalf("readInput() = (%q, %v), want (\"y\", nil)", got, err)
	}
}

func TestRunHappyPath(t *testing.T) {
	p := &fakeProvider{apiKey: "k", responses: []string{"```bash\nls -la\n```"}}
	app, out, _, _ := testApp(t, p, "list files", "q\n")
	if err := app.Run(); err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if len(p.requests) != 1 || p.requests[0] != "list files" {
		t.Errorf("requests = %q, want [\"list files\"]", p.requests)
	}
	if !strings.Contains(out.String(), "Here is the command --> ls -la <--") {
		t.Errorf("cleaned command not shown: %q", out.String())
	}
}

func TestRunRetriesAfterNotACommand(t *testing.T) {
	p := &fakeProvider{apiKey: "k", responses: []string{"not a command", "pwd"}}
	app, out, _, _ := testApp(t, p, "hello there", "where am i\nq\n")
	if err := app.Run(); err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if len(p.requests) != 2 || p.requests[1] != "where am i" {
		t.Errorf("requests = %q, want a retry with the new prompt", p.requests)
	}
	if !strings.Contains(out.String(), "Here is the command --> pwd <--") {
		t.Errorf("second command not shown: %q", out.String())
	}
}

func TestRunRefusesHiddenCharacters(t *testing.T) {
	p := &fakeProvider{apiKey: "k", responses: []string{"touch pwned #\rls -la", "pwd"}}
	app, out, _, _ := testApp(t, p, "list files", "try again\nq\n")
	if err := app.Run(); err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if !strings.Contains(out.String(), "control or invisible characters") || !strings.Contains(out.String(), `"touch pwned #\rls -la"`) {
		t.Errorf("hidden characters should be reported with the reply quoted: %q", out.String())
	}
	if strings.Contains(out.String(), "Here is the command --> touch") {
		t.Errorf("the reply must not be offered: %q", out.String())
	}
	if !strings.Contains(out.String(), "Here is the command --> pwd <--") {
		t.Errorf("the retry should be offered: %q", out.String())
	}
}

func TestRunQuitAfterNotACommand(t *testing.T) {
	p := &fakeProvider{apiKey: "k", responses: []string{"not a command"}}
	app, _, _, _ := testApp(t, p, "hello", "q\n")
	if err := app.Run(); err != nil {
		t.Fatalf("Run() = %v", err)
	}
	if len(p.requests) != 1 {
		t.Errorf("requests = %q, want exactly one", p.requests)
	}
}

func TestRunStopsWhenInputClosedAfterNotACommand(t *testing.T) {
	p := &fakeProvider{apiKey: "k", responses: []string{"not a command", "not a command"}}
	app, _, _, _ := testApp(t, p, "hello", "")
	err := app.Run()
	var inputErr *InputReadError
	if !errors.As(err, &inputErr) {
		t.Fatalf("Run() = %v, want *InputReadError instead of looping forever", err)
	}
	if len(p.requests) != 1 {
		t.Errorf("requests = %q, want exactly one", p.requests)
	}
}

func TestRunFollowUpRequests(t *testing.T) {
	p := &fakeProvider{apiKey: "k", responses: []string{"ls", "ls -a", "pwd"}}
	app, _, _, _ := testApp(t, p, "list", "r\ninclude hidden\nw\nwhere am i\nq\n")
	if err := app.Run(); err != nil {
		t.Fatalf("Run() = %v", err)
	}
	wantReq := []string{"list", "include hidden", "where am i"}
	if strings.Join(p.requests, "|") != strings.Join(wantReq, "|") {
		t.Errorf("requests = %q, want %q", p.requests, wantReq)
	}
	wantCtx := []bool{true, true, false}
	for i, want := range wantCtx {
		if i >= len(p.withContext) || p.withContext[i] != want {
			t.Fatalf("withContext = %v, want %v", p.withContext, wantCtx)
		}
	}
}

func TestRunNoAPIKey(t *testing.T) {
	t.Run("user declines", func(t *testing.T) {
		p := &fakeProvider{}
		app, out, _, _ := testApp(t, p, "list", "\n")
		if err := app.Run(); err != nil {
			t.Fatalf("Run() = %v", err)
		}
		if len(p.requests) != 0 {
			t.Errorf("no request should be sent without a key, got %q", p.requests)
		}
		if !strings.Contains(out.String(), "No API key was provided") {
			t.Errorf("missing exit message: %q", out.String())
		}
	})

	t.Run("user enters a key", func(t *testing.T) {
		p := &fakeProvider{responses: []string{"ls"}}
		app, _, _, _ := testApp(t, p, "list", "sk-typed\nq\n")
		if err := app.Run(); err != nil {
			t.Fatalf("Run() = %v", err)
		}
		if p.apiKey != "sk-typed" || len(p.requests) != 1 {
			t.Errorf("apiKey = %q, requests = %q", p.apiKey, p.requests)
		}
	})
}

func TestRunAPIErrors(t *testing.T) {
	t.Run("invalid key", func(t *testing.T) {
		apiErr := &OAIAPIError{StatusCode: 401, Code: "invalid_api_key"}
		p := &fakeProvider{
			apiKey: "bad",
			errs:   []error{apiErr},
			mapErr: func(err error) error { return &APIKeyError{OriginalError: err} },
		}
		app, _, errOut, _ := testApp(t, p, "list", "")
		err := app.Run()
		var keyErr *APIKeyError
		if !errors.As(err, &keyErr) {
			t.Fatalf("Run() = %v, want *APIKeyError", err)
		}
		if len(p.handled) != 1 || !errors.Is(p.handled[0], apiErr) {
			t.Errorf("provider should map the API error, handled = %v", p.handled)
		}
		if !strings.Contains(errOut.String(), "API key was rejected") {
			t.Errorf("missing invalid key hint: %q", errOut.String())
		}
	})

	t.Run("invalid key replaced at the prompt", func(t *testing.T) {
		p := &fakeProvider{
			apiKey:    "bad",
			errs:      []error{&OAIAPIError{StatusCode: 401, Code: "invalid_api_key"}},
			responses: []string{"", "ls"},
			mapErr: func(err error) error {
				var apiErr *OAIAPIError
				if errors.As(err, &apiErr) {
					return &APIKeyError{OriginalError: err}
				}
				return err
			},
		}
		app, out, errOut, _ := testApp(t, p, "list", "sk-good\nq\n")
		if err := app.Run(); err != nil {
			t.Fatalf("Run() = %v", err)
		}
		if p.apiKey != "sk-good" || len(p.requests) != 2 {
			t.Errorf("apiKey = %q, requests = %q, want the request retried with the new key", p.apiKey, p.requests)
		}
		if !strings.Contains(errOut.String(), "API key was rejected") || !strings.Contains(out.String(), "Here is the command --> ls <--") {
			t.Errorf("stdout %q, stderr %q", out.String(), errOut.String())
		}
	})

	t.Run("other error", func(t *testing.T) {
		boom := errors.New("connection refused")
		p := &fakeProvider{apiKey: "k", errs: []error{boom}}
		app, _, errOut, _ := testApp(t, p, "list", "")
		if err := app.Run(); !errors.Is(err, boom) {
			t.Fatalf("Run() = %v, want %v", err, boom)
		}
		if errOut.Len() != 0 {
			t.Errorf("no key hint expected for other errors, got %q", errOut.String())
		}
	})
}
