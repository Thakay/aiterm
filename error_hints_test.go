package main

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"testing"
)

func TestReportRuntimeErrorHintsForAPIResponses(t *testing.T) {
	tests := []struct {
		name      string
		status    int
		errorType string
		code      string
		wantHint  string
	}{
		{
			name:      "not found status",
			status:    404,
			errorType: "invalid_request_error",
			code:      "not_found",
			wantHint:  "The model is wrong or not available on this endpoint; check -model or AITERM_MODEL.",
		},
		{
			name:      "model not found code",
			status:    400,
			errorType: "invalid_request_error",
			code:      "model_not_found",
			wantHint:  "The model is wrong or not available on this endpoint; check -model or AITERM_MODEL.",
		},
		{
			name:      "quota exhausted",
			status:    429,
			errorType: "insufficient_quota",
			code:      "insufficient_quota",
			wantHint:  "The account has no credit left; check billing.",
		},
		{
			name:      "rate limited",
			status:    429,
			errorType: "rate_limit_error",
			code:      "rate_limit_exceeded",
			wantHint:  "Rate limited; wait a moment and retry.",
		},
		{
			name:      "server error lower bound",
			status:    500,
			errorType: "server_error",
			code:      "internal_error",
			wantHint:  "The service had a problem; retry later.",
		},
		{
			name:      "server error upper bound",
			status:    599,
			errorType: "server_error",
			code:      "internal_error",
			wantHint:  "The service had a problem; retry later.",
		},
		{
			name:      "other API error",
			status:    400,
			errorType: "invalid_request_error",
			code:      "invalid_request_error",
		},
		{
			name:      "rejected API key keeps existing handling",
			status:    401,
			errorType: "invalid_request_error",
			code:      "invalid_api_key",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body := fmt.Sprintf(
				"{\"error\":{\"message\":%q,\"type\":%q,\"code\":%q}}",
				"request rejected", tt.errorType, tt.code,
			)
			srv := newChatServer(t, tt.status, body)
			var out, errOut bytes.Buffer
			err := run(
				[]string{"-url", srv.URL, "-key", "sk-test", "list files"},
				envFrom(nil), strings.NewReader("\n"), &out, &errOut,
			)
			if err == nil {
				t.Fatal("run() error = nil, want API error")
			}

			reportRuntimeError(&errOut, err)
			got := errOut.String()
			if want := "aiterm: " + err.Error() + "\n"; !strings.Contains(got, want) {
				t.Errorf("stderr = %q, want unchanged raw error line %q", got, want)
			}
			if tt.wantHint == "" {
				if strings.Contains(got, "hint: ") {
					t.Errorf("stderr = %q, want no hint", got)
				}
			} else if want := "hint: " + tt.wantHint + "\n"; !strings.HasSuffix(got, want) {
				t.Errorf("stderr = %q, want final hint line %q", got, want)
			}
			if tt.status == 401 && !strings.Contains(got, "The API key was rejected.") {
				t.Errorf("stderr = %q, want the existing invalid-key handling", got)
			}
		})
	}
}

func TestReportRuntimeErrorHintForClosedLoopbackServer(t *testing.T) {
	srv := newChatServer(t, 200, successJSON("ls"))
	endpoint := srv.URL
	srv.Close()

	var out, errOut bytes.Buffer
	err := run(
		[]string{"-url", endpoint, "-key", "sk-test", "-timeout", "3s", "list files"},
		envFrom(nil), strings.NewReader(""), &out, &errOut,
	)
	if err == nil {
		t.Fatal("run() error = nil, want connection refused")
	}
	if !errors.Is(err, syscall.ECONNREFUSED) {
		t.Fatalf("run() error = %v, want connection refused", err)
	}

	reportRuntimeError(&errOut, err)
	got := errOut.String()
	if want := "aiterm: " + err.Error() + "\n"; !strings.Contains(got, want) {
		t.Errorf("stderr = %q, want unchanged raw error line %q", got, want)
	}
	wantHint := "hint: The local server (Ollama or LM Studio) is not running.\n"
	if !strings.HasSuffix(got, wantHint) {
		t.Errorf("stderr = %q, want final hint line %q", got, wantHint)
	}
}

func TestRuntimeErrorHintOnlyForLoopbackConnectionRefused(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		wantHint bool
	}{
		{
			name:     "localhost",
			endpoint: "http://localhost:11434/v1/chat/completions",
			wantHint: true,
		},
		{
			name:     "IPv4 loopback",
			endpoint: "http://127.0.0.1:11434/v1/chat/completions",
			wantHint: true,
		},
		{
			name:     "IPv6 loopback",
			endpoint: "http://[::1]:11434/v1/chat/completions",
			wantHint: true,
		},
		{
			name:     "remote HTTPS",
			endpoint: "https://api.example.com/v1/chat/completions",
			wantHint: false,
		},
		{
			name:     "remote IPv4",
			endpoint: "http://192.0.2.1:11434/v1/chat/completions",
			wantHint: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := &ExecutionError{OriginalError: &url.Error{
				Op: "Post", URL: tt.endpoint, Err: syscall.ECONNREFUSED,
			}}
			got := runtimeErrorHint(err)
			if tt.wantHint && got == "" {
				t.Error("runtimeErrorHint() = empty, want local-server hint")
			}
			if !tt.wantHint && got != "" {
				t.Errorf("runtimeErrorHint() = %q, want no hint", got)
			}
		})
	}
}

func TestMainReportsRuntimeHintAndKeepsExitCode(t *testing.T) {
	if os.Getenv("AITERM_TEST_MAIN_ERROR") == "1" {
		os.Args = []string{"aiterm", "list files"}
		main()
		return
	}

	srv := newChatServer(t, 500, `{"error":{"message":"request rejected","type":"server_error","code":"internal_error"}}`)
	cmd := exec.Command(os.Args[0], "-test.run=^TestMainReportsRuntimeHintAndKeepsExitCode$")
	cmd.Env = make([]string, 0, len(os.Environ())+3)
	for _, entry := range os.Environ() {
		if strings.HasPrefix(entry, "AITERM_TEST_MAIN_ERROR=") ||
			strings.HasPrefix(entry, "AITERM_URL=") ||
			strings.HasPrefix(entry, "AITERM_MODEL=") ||
			strings.HasPrefix(entry, "AITERM_TIMEOUT=") ||
			strings.HasPrefix(entry, "OPENAI_KEY=") {
			continue
		}
		cmd.Env = append(cmd.Env, entry)
	}
	cmd.Env = append(cmd.Env,
		"AITERM_TEST_MAIN_ERROR=1",
		"AITERM_URL="+srv.URL,
		"OPENAI_KEY=sk-test",
	)

	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("exit = %v, want status 1; output: %s", err, output)
	}
	if want := "aiterm: API error (status 500, server_error): request rejected (code internal_error)\n"; !strings.Contains(string(output), want) {
		t.Errorf("output = %q, want unchanged raw error line %q", output, want)
	}
	if want := "hint: The service had a problem; retry later.\n"; !strings.HasSuffix(string(output), want) {
		t.Errorf("output = %q, want final hint line %q", output, want)
	}
}
