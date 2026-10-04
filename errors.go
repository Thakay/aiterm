package main

import "fmt"

type MarshalingError struct {
	OriginalError error
}

func (e *MarshalingError) Error() string {
	return fmt.Sprintf("failed marshaling payload: %v", e.OriginalError)
}

func (e *MarshalingError) Unwrap() error { return e.OriginalError }

type UnMarshalingError struct {
	OriginalError error
}

func (e *UnMarshalingError) Error() string {
	return fmt.Sprintf("failed unmarshaling response: %v", e.OriginalError)
}

func (e *UnMarshalingError) Unwrap() error { return e.OriginalError }

type RequestCreationError struct {
	OriginalError error
}

func (e *RequestCreationError) Error() string {
	return fmt.Sprintf("failed creating request: %v", e.OriginalError)
}

func (e *RequestCreationError) Unwrap() error { return e.OriginalError }

type ExecutionError struct {
	OriginalError error
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("failed executing request: %v", e.OriginalError)
}

func (e *ExecutionError) Unwrap() error { return e.OriginalError }

type ResponseReadError struct {
	OriginalError error
}

func (e *ResponseReadError) Error() string {
	return fmt.Sprintf("failed reading response body: %v", e.OriginalError)
}

func (e *ResponseReadError) Unwrap() error { return e.OriginalError }

type OAIAPIError struct {
	StatusCode int
	Type       string
	Message    string
	Code       string
}

func (e *OAIAPIError) Error() string {
	return fmt.Sprintf("API error (status %d): %s - %s - %s", e.StatusCode, e.Type, e.Message, e.Code)
}

type InputReadError struct {
	OriginalError error
}

func (e *InputReadError) Error() string {
	return fmt.Sprintf("failed reading Input from terminal: %v", e.OriginalError)
}

func (e *InputReadError) Unwrap() error { return e.OriginalError }

type APIKeyError struct {
	OriginalError error
}

func (e *APIKeyError) Error() string {
	return fmt.Sprintf("invalid API key: %v", e.OriginalError)
}

func (e *APIKeyError) Unwrap() error { return e.OriginalError }
