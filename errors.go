package main

import "fmt"

type MarshalingError struct {
	OriginalError error
}

func (e *MarshalingError) Error() string {
	return fmt.Sprintf("failed marshaling payload: %v", e.OriginalError)
}

type UnMarshalingError struct {
	OriginalError error
}

func (e *UnMarshalingError) Error() string {
	return fmt.Sprintf("failed marshaling payload: %v", e.OriginalError)
}

type RequestCreationError struct {
	OriginalError error
}

func (e *RequestCreationError) Error() string {
	return fmt.Sprintf("failed creating request: %v", e.OriginalError)
}

type ExecutionError struct {
	OriginalError error
}

func (e *ExecutionError) Error() string {
	return fmt.Sprintf("failed executing request: %v", e.OriginalError)
}

type ResponseReadError struct {
	OriginalError error
}

func (e *ResponseReadError) Error() string {
	return fmt.Sprintf("failed reading response body: %v", e.OriginalError)
}

type APIError struct {
	Type    string
	Message string
	Code    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error: %s - %s - %s", e.Type, e.Message, e.Code)
}

type InputReadError struct {
	OriginalError error
}

func (e *InputReadError) Error() string {
	return fmt.Sprintf("failed reading Input from terminal: %v", e.OriginalError)
}
