package main

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func envFrom(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestParseArgs(t *testing.T) {
	tests := []struct {
		name string
		args []string
		env  map[string]string
		want config
	}{
		{
			name: "defaults",
			args: []string{"list files"},
			want: config{url: defaultEndPoint, model: defaultModel, prompt: "list files"},
		},
		{
			name: "environment",
			args: []string{"list files"},
			env:  map[string]string{varKeyName: "sk-env", varURLName: "http://localhost:11434/v1/chat/completions", varModelName: "llama3"},
			want: config{apiKey: "sk-env", url: "http://localhost:11434/v1/chat/completions", model: "llama3", prompt: "list files"},
		},
		{
			name: "flags win over environment",
			args: []string{"-key", "sk-flag", "-url", "http://example.test", "-model", "gpt-flag", "list files"},
			env:  map[string]string{varKeyName: "sk-env", varURLName: "http://env.test", varModelName: "gpt-env"},
			want: config{apiKey: "sk-flag", url: "http://example.test", model: "gpt-flag", prompt: "list files"},
		},
		{
			name: "unquoted prompt is joined",
			args: []string{"find", "all", "go", "files"},
			want: config{url: defaultEndPoint, model: defaultModel, prompt: "find all go files"},
		},
		{
			name: "no prompt",
			args: []string{"-key", "sk"},
			want: config{apiKey: "sk", url: defaultEndPoint, model: defaultModel},
		},
		{
			name: "version flag",
			args: []string{"-version"},
			want: config{url: defaultEndPoint, model: defaultModel, showVersion: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			got, err := parseArgs(tt.args, envFrom(tt.env), &out)
			if err != nil {
				t.Fatalf("parseArgs() error = %v (output %q)", err, out.String())
			}
			if *got != tt.want {
				t.Errorf("parseArgs() = %+v, want %+v", *got, tt.want)
			}
		})
	}
}

func TestParseArgsErrors(t *testing.T) {
	var out bytes.Buffer
	if _, err := parseArgs([]string{"-h"}, envFrom(nil), &out); !errors.Is(err, flag.ErrHelp) {
		t.Errorf("-h: error = %v, want flag.ErrHelp", err)
	}
	if !strings.Contains(out.String(), "Usage: aiterm") {
		t.Errorf("-h should print usage, got %q", out.String())
	}

	out.Reset()
	if _, err := parseArgs([]string{"-nope"}, envFrom(nil), &out); err == nil {
		t.Error("unknown flag should be an error")
	}
}

func TestHelpDoesNotLeakAPIKey(t *testing.T) {
	var out, errOut bytes.Buffer
	env := envFrom(map[string]string{varKeyName: "sk-super-secret"})
	if err := run([]string{"-h"}, env, strings.NewReader(""), &out, &errOut); err != nil {
		t.Fatalf("run(-h) = %v", err)
	}
	if strings.Contains(out.String()+errOut.String(), "sk-super-secret") {
		t.Errorf("help output leaks the API key: %q", errOut.String())
	}
}

func TestRunVersion(t *testing.T) {
	var out, errOut bytes.Buffer
	if err := run([]string{"-version"}, envFrom(nil), strings.NewReader(""), &out, &errOut); err != nil {
		t.Fatalf("run(-version) = %v", err)
	}
	if !strings.HasPrefix(out.String(), "aiterm ") {
		t.Errorf("version output = %q", out.String())
	}
}

func TestVersionString(t *testing.T) {
	oldVersion, oldCommit, oldDate := version, commit, date
	t.Cleanup(func() { version, commit, date = oldVersion, oldCommit, oldDate })

	version, commit, date = "0.2.0", "abc1234", "2026-10-03T00:00:00Z"
	if got, want := versionString(), "aiterm 0.2.0 (commit abc1234, built 2026-10-03T00:00:00Z)"; got != want {
		t.Errorf("versionString() = %q, want %q", got, want)
	}

	version, commit = "dev", "none"
	if got := versionString(); !strings.HasPrefix(got, "aiterm ") || strings.Contains(got, "commit") {
		t.Errorf("versionString() for a dev build = %q", got)
	}
}

func TestRunUsageErrors(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantOut string
	}{
		{"no prompt", nil, "No prompt was provided."},
		{"blank prompt", []string{"   "}, "No prompt was provided."},
		{"unknown flag", []string{"-nope", "list"}, "flag provided but not defined"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out, errOut bytes.Buffer
			err := run(tt.args, envFrom(nil), strings.NewReader(""), &out, &errOut)
			if !errors.Is(err, errUsage) {
				t.Fatalf("run() = %v, want errUsage", err)
			}
			if !strings.Contains(errOut.String(), tt.wantOut) {
				t.Errorf("stderr = %q, want it to contain %q", errOut.String(), tt.wantOut)
			}
		})
	}
}

func TestRunEndToEnd(t *testing.T) {
	srv := newChatServer(t, 200, successJSON("ls -la"))
	var out, errOut bytes.Buffer
	args := []string{"-url", srv.URL, "-key", "sk-test", "-model", "gpt-test", "list", "all", "files"}
	if err := run(args, envFrom(nil), strings.NewReader("q\n"), &out, &errOut); err != nil {
		t.Fatalf("run() = %v (stderr %q)", err, errOut.String())
	}

	if !strings.Contains(out.String(), "Here is the command --> ls -la <--") {
		t.Errorf("command not shown: %q", out.String())
	}
	if got := srv.headers[0].Get("Authorization"); got != "Bearer sk-test" {
		t.Errorf("Authorization = %q", got)
	}
	body := srv.lastBody(t)
	if body["model"] != "gpt-test" {
		t.Errorf("model = %v, want gpt-test", body["model"])
	}
	msgs := messagesOf(t, body)
	if last := msgs[len(msgs)-1]; last != [2]string{"user", "list all files"} {
		t.Errorf("last message = %v", last)
	}
}

func TestRunEndToEndConfiguredByEnvironment(t *testing.T) {
	srv := newChatServer(t, 200, successJSON("pwd"))
	env := envFrom(map[string]string{varKeyName: "sk-env", varURLName: srv.URL, varModelName: "gpt-env"})
	var out, errOut bytes.Buffer
	if err := run([]string{"where am i"}, env, strings.NewReader("q\n"), &out, &errOut); err != nil {
		t.Fatalf("run() = %v (stderr %q)", err, errOut.String())
	}
	if got := srv.headers[0].Get("Authorization"); got != "Bearer sk-env" {
		t.Errorf("Authorization = %q", got)
	}
	if body := srv.lastBody(t); body["model"] != "gpt-env" {
		t.Errorf("model = %v, want gpt-env", body["model"])
	}
}

func TestRunEndToEndInvalidKey(t *testing.T) {
	srv := newChatServer(t, 401, `{"error":{"message":"Incorrect API key provided","type":"invalid_request_error","param":null,"code":"invalid_api_key"}}`)
	var out, errOut bytes.Buffer
	err := run([]string{"-url", srv.URL, "-key", "sk-bad", "list"}, envFrom(nil), strings.NewReader(""), &out, &errOut)
	var keyErr *APIKeyError
	if !errors.As(err, &keyErr) {
		t.Fatalf("run() = %v, want *APIKeyError", err)
	}
	if !strings.Contains(errOut.String(), "API key was rejected") {
		t.Errorf("stderr = %q", errOut.String())
	}
}

// TestMainExitCodes runs main() in a subprocess to check the exit codes the
// shell sees.
func TestMainExitCodes(t *testing.T) {
	if os.Getenv("AITERM_RUN_MAIN") == "1" {
		os.Args = append([]string{"aiterm"}, strings.Fields(os.Getenv("AITERM_ARGS"))...)
		main()
		os.Exit(0)
	}

	tests := []struct {
		args string
		want int
	}{
		{"-version", 0},
		{"-h", 0},
		{"", 2},
		{"-nope", 2},
		// Nothing listens on port 1, so the request fails right away.
		{"-url http://127.0.0.1:1 -key sk-test list files", 1},
	}
	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestMainExitCodes$")
			cmd.Env = append(os.Environ(), "AITERM_RUN_MAIN=1", "AITERM_ARGS="+tt.args)
			err := cmd.Run()
			code := 0
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				code = exitErr.ExitCode()
			} else if err != nil {
				t.Fatalf("running main: %v", err)
			}
			if code != tt.want {
				t.Errorf("exit code = %d, want %d", code, tt.want)
			}
		})
	}
}
