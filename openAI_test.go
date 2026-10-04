package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func newTestProvider(t *testing.T, url string) *OpenAIProvider {
	t.Helper()
	p, ok := NewOpenAIProvider("sk-test", &OpenAIOptions{
		ProviderOptions: &ProviderOptions{URL: url},
		model:           "gpt-test",
		withContext:     true,
	}).(*OpenAIProvider)
	if !ok {
		t.Fatal("NewOpenAIProvider did not return an *OpenAIProvider")
	}
	return p
}

func TestNewOpenAIProviderDefaults(t *testing.T) {
	p := NewOpenAIProvider("sk-test", nil).(*OpenAIProvider)
	if p.options.URL != defaultEndPoint {
		t.Errorf("URL = %q, want %q", p.options.URL, defaultEndPoint)
	}
	if p.options.APIKey != "sk-test" {
		t.Errorf("APIKey = %q", p.options.APIKey)
	}
	if p.options.model != defaultModel {
		t.Errorf("model = %q, want %q", p.options.model, defaultModel)
	}
	if len(p.options.messages) != 1 || p.options.messages[0]["role"] != "system" {
		t.Errorf("messages = %v, want a single system prompt", p.options.messages)
	}
	if p.httpClient == nil || p.httpClient.Timeout != defaultTimeout {
		t.Errorf("http client should time out after %v", defaultTimeout)
	}
}

func TestNewOpenAIProviderTimeout(t *testing.T) {
	p := NewOpenAIProvider("sk", &OpenAIOptions{timeout: 5 * time.Second}).(*OpenAIProvider)
	if p.httpClient.Timeout != 5*time.Second {
		t.Errorf("timeout = %v, want 5s", p.httpClient.Timeout)
	}
}

func TestNewOpenAIProviderFillsMissingOptions(t *testing.T) {
	p := NewOpenAIProvider("sk-a", &OpenAIOptions{}).(*OpenAIProvider)
	if p.options.ProviderOptions == nil || p.options.URL != defaultEndPoint || p.options.APIKey != "sk-a" {
		t.Errorf("provider options not defaulted: %+v", p.options.ProviderOptions)
	}
	if p.options.model != defaultModel || p.options.messages == nil {
		t.Errorf("model/messages not defaulted: %q %v", p.options.model, p.options.messages)
	}

	p = NewOpenAIProvider("sk-b", &OpenAIOptions{ProviderOptions: &ProviderOptions{APIKey: "ignored"}}).(*OpenAIProvider)
	if p.options.URL != defaultEndPoint || p.options.APIKey != "sk-b" {
		t.Errorf("explicit key argument should win and URL default: %+v", p.options.ProviderOptions)
	}
}

func TestSystemPrompt(t *testing.T) {
	tests := map[string]string{"darwin": "macOS", "linux": "Linux", "freebsd": "FreeBSD", "plan9": "plan9"}
	for goos, want := range tests {
		if got := osName(goos); got != want {
			t.Errorf("osName(%q) = %q, want %q", goos, got, want)
		}
		if prompt := systemPrompt(goos); !strings.Contains(prompt, want) || !strings.Contains(prompt, "not a command") {
			t.Errorf("systemPrompt(%q) = %q", goos, prompt)
		}
	}
}

func TestFetchSuccess(t *testing.T) {
	srv := newChatServer(t, 200, successJSON("ls -la"))
	p := newTestProvider(t, srv.URL)

	got, err := p.fetch("list files")
	if err != nil {
		t.Fatalf("fetch() error = %v", err)
	}
	if got != "ls -la" {
		t.Errorf("fetch() = %q, want %q", got, "ls -la")
	}

	h := srv.headers[0]
	if h.Get("Authorization") != "Bearer sk-test" || h.Get("Content-Type") != "application/json" {
		t.Errorf("unexpected headers: %v", h)
	}

	body := srv.lastBody(t)
	if body["model"] != "gpt-test" {
		t.Errorf("payload model = %v, want gpt-test", body["model"])
	}
	// Reasoning models reject max_tokens and custom sampling parameters, so
	// none are sent and every model works.
	for _, key := range []string{"max_tokens", "temperature", "top_p", "frequency_penalty", "presence_penalty"} {
		if _, ok := body[key]; ok {
			t.Errorf("payload should not contain %q: %v", key, body)
		}
	}
	msgs := messagesOf(t, body)
	if len(msgs) != 2 || msgs[0][0] != "system" || msgs[1] != [2]string{"user", "list files"} {
		t.Errorf("messages = %v", msgs)
	}
}

func TestFetchKeepsContext(t *testing.T) {
	srv := newChatServer(t, 200, successJSON("ls"))
	p := newTestProvider(t, srv.URL)

	if _, err := p.fetch("list files", p.newFetchConfig(true)); err != nil {
		t.Fatal(err)
	}
	if _, err := p.fetch("include hidden ones", p.newFetchConfig(true)); err != nil {
		t.Fatal(err)
	}
	msgs := messagesOf(t, srv.lastBody(t))
	want := [][2]string{{"user", "list files"}, {"assistant", "ls"}, {"user", "include hidden ones"}}
	if len(msgs) != 4 {
		t.Fatalf("messages = %v, want system + %v", msgs, want)
	}
	for i, w := range want {
		if msgs[i+1] != w {
			t.Errorf("message %d = %v, want %v", i+1, msgs[i+1], w)
		}
	}
}

func TestFetchWithoutContext(t *testing.T) {
	srv := newChatServer(t, 200, successJSON("ls"))
	p := newTestProvider(t, srv.URL)

	if _, err := p.fetch("list files"); err != nil {
		t.Fatal(err)
	}
	if _, err := p.fetch("where am i", p.newFetchConfig(false)); err != nil {
		t.Fatal(err)
	}
	msgs := messagesOf(t, srv.lastBody(t))
	if len(msgs) != 2 || msgs[1] != [2]string{"user", "where am i"} {
		t.Errorf("messages = %v, want only the system prompt and the new request", msgs)
	}
}

func TestFetchAPIError(t *testing.T) {
	srv := newChatServer(t, 401, `{"error":{"message":"Incorrect API key provided","type":"invalid_request_error","param":null,"code":"invalid_api_key"}}`)
	p := newTestProvider(t, srv.URL)

	_, err := p.fetch("list files")
	var apiErr *OAIAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("fetch() error = %v, want *OAIAPIError", err)
	}
	want := OAIAPIError{StatusCode: 401, Type: "invalid_request_error", Message: "Incorrect API key provided", Code: "invalid_api_key"}
	if *apiErr != want {
		t.Errorf("error = %+v, want %+v", *apiErr, want)
	}
	if len(p.options.messages) != 1 {
		t.Errorf("failed request should not be kept in the history: %v", p.options.messages)
	}
}

func TestFetchNonJSONError(t *testing.T) {
	srv := newChatServer(t, 502, "<html>upstream down</html>")
	p := newTestProvider(t, srv.URL)

	_, err := p.fetch("list files")
	var apiErr *OAIAPIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("fetch() error = %v, want *OAIAPIError", err)
	}
	if apiErr.StatusCode != 502 || apiErr.Type != "http_error" || !strings.Contains(apiErr.Message, "Bad Gateway") || !strings.Contains(apiErr.Message, "upstream down") {
		t.Errorf("error = %+v", *apiErr)
	}
}

func TestFetchMalformedSuccess(t *testing.T) {
	srv := newChatServer(t, 200, "not json")
	p := newTestProvider(t, srv.URL)
	_, err := p.fetch("list files")
	var unmarshalErr *UnMarshalingError
	if !errors.As(err, &unmarshalErr) {
		t.Fatalf("fetch() error = %v, want *UnMarshalingError", err)
	}
}

func TestFetchNoChoices(t *testing.T) {
	srv := newChatServer(t, 200, `{"choices":[]}`)
	p := newTestProvider(t, srv.URL)
	if _, err := p.fetch("list files"); err == nil || !strings.Contains(err.Error(), "choices") {
		t.Fatalf("fetch() error = %v, want a missing choices error", err)
	}
}

func TestFetchTimeout(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		<-release
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) }) // runs first, so Close does not wait

	p := NewOpenAIProvider("sk", &OpenAIOptions{ProviderOptions: &ProviderOptions{URL: srv.URL}, timeout: 50 * time.Millisecond}).(*OpenAIProvider)
	_, err := p.fetch("list files")
	var execErr *ExecutionError
	if !errors.As(err, &execErr) || !strings.Contains(err.Error(), "-timeout") {
		t.Fatalf("fetch() error = %v, want an *ExecutionError that explains how to raise the timeout", err)
	}
}

func TestFetchResponseTooLarge(t *testing.T) {
	srv := newChatServer(t, 200, `{"choices":[],"padding":"`+strings.Repeat("a", maxResponseBytes)+`"}`)
	p := newTestProvider(t, srv.URL)
	_, err := p.fetch("list files")
	var readErr *ResponseReadError
	if !errors.As(err, &readErr) || !strings.Contains(err.Error(), "larger than") {
		t.Fatalf("fetch() error = %v, want a *ResponseReadError about the size", err)
	}
}

func TestFetchRequestErrors(t *testing.T) {
	t.Run("invalid URL", func(t *testing.T) {
		p := newTestProvider(t, "://not-a-url")
		_, err := p.fetch("list files")
		var reqErr *RequestCreationError
		if !errors.As(err, &reqErr) {
			t.Fatalf("fetch() error = %v, want *RequestCreationError", err)
		}
	})

	t.Run("server unreachable", func(t *testing.T) {
		srv := newChatServer(t, 200, successJSON("ls"))
		url := srv.URL
		srv.Close()
		p := newTestProvider(t, url)
		_, err := p.fetch("list files")
		var execErr *ExecutionError
		if !errors.As(err, &execErr) {
			t.Fatalf("fetch() error = %v, want *ExecutionError", err)
		}
	})
}

func TestHasAPIKey(t *testing.T) {
	if NewOpenAIProvider("", nil).hasAPIKey() {
		t.Error("hasAPIKey() = true for an empty key")
	}
	p := NewOpenAIProvider("", nil)
	p.setAPIKey("sk-test")
	if !p.hasAPIKey() {
		t.Error("hasAPIKey() = false after setAPIKey")
	}
}

func TestHandleAPIError(t *testing.T) {
	p := NewOpenAIProvider("sk", nil)

	for _, apiErr := range []*OAIAPIError{
		{StatusCode: 401, Code: "invalid_api_key"},
		{StatusCode: 401},
		{StatusCode: 400, Code: "invalid_api_key"},
	} {
		var keyErr *APIKeyError
		if err := p.handleAPIError(apiErr); !errors.As(err, &keyErr) || !errors.Is(err, apiErr) {
			t.Errorf("handleAPIError(%+v) = %v, want *APIKeyError wrapping it", *apiErr, err)
		}
	}

	rateLimited := &OAIAPIError{StatusCode: http.StatusTooManyRequests, Code: "rate_limit_exceeded"}
	other := errors.New("boom")
	for _, in := range []error{rateLimited, other} {
		err := p.handleAPIError(in)
		var keyErr *APIKeyError
		if !errors.Is(err, in) || errors.As(err, &keyErr) {
			t.Errorf("handleAPIError(%v) = %v, want it unchanged", in, err)
		}
	}
}

func TestNewFetchConfigIgnoresOtherProviders(_ *testing.T) {
	p := NewOpenAIProvider("sk", nil)
	// Must not panic or exit when applied to an unrelated value.
	p.newFetchConfig(false)(&fakeProvider{})
}

func TestTruncate(t *testing.T) {
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate() = %q", got)
	}
	if got := truncate("héllo wörld", 5); got != "héllo..." {
		t.Errorf("truncate() = %q, want rune safe cut", got)
	}
}
