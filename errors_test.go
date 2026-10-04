package main

import (
	"errors"
	"testing"
)

func TestErrorMessagesAndUnwrap(t *testing.T) {
	inner := errors.New("inner cause")
	tests := []struct {
		err  error
		want string
	}{
		{&MarshalingError{inner}, "failed marshaling payload: inner cause"},
		{&UnMarshalingError{inner}, "failed unmarshaling response: inner cause"},
		{&RequestCreationError{inner}, "failed creating request: inner cause"},
		{&ExecutionError{inner}, "failed executing request: inner cause"},
		{&ResponseReadError{inner}, "failed reading response body: inner cause"},
		{&InputReadError{inner}, "failed reading Input from terminal: inner cause"},
		{&APIKeyError{inner}, "invalid API key: inner cause"},
	}
	for _, tt := range tests {
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("%T.Error() = %q, want %q", tt.err, got, tt.want)
		}
		if !errors.Is(tt.err, inner) {
			t.Errorf("%T should unwrap to its original error", tt.err)
		}
	}
}

func TestOAIAPIErrorMessage(t *testing.T) {
	tests := []struct {
		err  *OAIAPIError
		want string
	}{
		{
			&OAIAPIError{StatusCode: 429, Type: "requests", Message: "Rate limit reached", Code: "rate_limit_exceeded"},
			"API error (status 429, requests): Rate limit reached (code rate_limit_exceeded)",
		},
		{&OAIAPIError{StatusCode: 502, Type: "http_error", Message: "Bad Gateway"}, "API error (status 502, http_error): Bad Gateway"},
		{&OAIAPIError{StatusCode: 500}, "API error (status 500)"},
	}
	for _, tt := range tests {
		if got := tt.err.Error(); got != tt.want {
			t.Errorf("Error() = %q, want %q", got, tt.want)
		}
	}
}
