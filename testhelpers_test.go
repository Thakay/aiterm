package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeProvider is a scripted APIProvider used to drive App without the network.
type fakeProvider struct {
	responses   []string
	errs        []error
	requests    []string
	withContext []bool
	apiKey      string
	handled     []error
	mapErr      func(error) error
}

func (f *fakeProvider) fetch(userRequest string, opts ...FetchConfig) (string, error) {
	for _, opt := range opts {
		opt(f)
	}
	f.requests = append(f.requests, userRequest)
	i := len(f.requests) - 1
	if i < len(f.errs) && f.errs[i] != nil {
		return "", f.errs[i]
	}
	if i < len(f.responses) {
		return f.responses[i], nil
	}
	return "", errors.New("fakeProvider: no scripted response left")
}

func (f *fakeProvider) hasAPIKey() bool { return f.apiKey != "" }

func (f *fakeProvider) handleAPIError(err error) error {
	f.handled = append(f.handled, err)
	if f.mapErr != nil {
		return f.mapErr(err)
	}
	return err
}

func (f *fakeProvider) newFetchConfig(withCtxt bool) FetchConfig {
	return func(o interface{}) {
		if p, ok := o.(*fakeProvider); ok {
			p.withContext = append(p.withContext, withCtxt)
		}
	}
}

func (f *fakeProvider) setAPIKey(apikey string) { f.apiKey = apikey }

// testApp builds an App wired to in-memory streams and a fake clipboard.
func testApp(t *testing.T, provider APIProvider, request, input string) (*App, *bytes.Buffer, *bytes.Buffer, *[]string) {
	t.Helper()
	var out, errOut bytes.Buffer
	app := newApp(provider, request, strings.NewReader(input), &out, &errOut)
	copied := &[]string{}
	app.copyToClipboard = func(s string) error {
		*copied = append(*copied, s)
		return nil
	}
	return app, &out, &errOut, copied
}

// chatServer is an httptest server that mimics the chat completions endpoint.
type chatServer struct {
	*httptest.Server
	mu       sync.Mutex
	bodies   []map[string]interface{}
	headers  []http.Header
	status   int
	response string
}

func newChatServer(t *testing.T, status int, response string) *chatServer {
	t.Helper()
	s := &chatServer{status: status, response: response}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading request body: %v", err)
		}
		var body map[string]interface{}
		if err := json.Unmarshal(raw, &body); err != nil {
			t.Errorf("request body is not JSON: %v", err)
		}
		s.mu.Lock()
		s.bodies = append(s.bodies, body)
		s.headers = append(s.headers, r.Header.Clone())
		status, response := s.status, s.response
		s.mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = io.WriteString(w, response)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *chatServer) lastBody(t *testing.T) map[string]interface{} {
	t.Helper()
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.bodies) == 0 {
		t.Fatal("no request reached the server")
	}
	return s.bodies[len(s.bodies)-1]
}

func successJSON(content string) string {
	b, _ := json.Marshal(map[string]interface{}{
		"id":     "chatcmpl-test",
		"object": "chat.completion",
		"model":  "gpt-test",
		"choices": []map[string]interface{}{
			{"index": 0, "message": map[string]string{"role": "assistant", "content": content}, "finish_reason": "stop"},
		},
	})
	return string(b)
}

// messagesOf extracts the role/content pairs from a decoded request body.
func messagesOf(t *testing.T, body map[string]interface{}) [][2]string {
	t.Helper()
	raw, ok := body["messages"].([]interface{})
	if !ok {
		t.Fatalf("messages missing or wrong type: %#v", body["messages"])
	}
	var msgs [][2]string
	for _, m := range raw {
		mm := m.(map[string]interface{})
		msgs = append(msgs, [2]string{mm["role"].(string), mm["content"].(string)})
	}
	return msgs
}
