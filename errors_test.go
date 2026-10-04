package main

import (
	"errors"
	"strings"
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
	err := &OAIAPIError{StatusCode: 429, Type: "requests", Message: "Rate limit reached", Code: "rate_limit_exceeded"}
	for _, part := range []string{"429", "requests", "Rate limit reached", "rate_limit_exceeded"} {
		if !strings.Contains(err.Error(), part) {
			t.Errorf("Error() = %q, missing %q", err.Error(), part)
		}
	}
}
