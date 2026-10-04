package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"runtime"
	"time"
)

const (
	defaultEndPoint = "https://api.openai.com/v1/chat/completions"
	defaultModel    = "gpt-4.1-mini"
	defaultTimeout  = 2 * time.Minute

	// maxResponseBytes bounds how much of a response is read. A reply is one
	// short command, so anything near this size is a misbehaving endpoint.
	maxResponseBytes = 1 << 20
)

type ErrorResponse struct {
	Error struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Param   interface{} `json:"param"`
		Code    string      `json:"code"`
	} `json:"error"`
}

// SuccessResponse struct to match the JSON structure of the API response
type SuccessResponse struct {
	ID                string      `json:"id"`
	Object            string      `json:"object"`
	Created           int         `json:"created"`
	Model             string      `json:"model"`
	Choices           []Choice    `json:"choices"`
	Usage             Usage       `json:"usage"`
	SystemFingerprint interface{} `json:"system_fingerprint"`
}

type Choice struct {
	Index        int         `json:"index"`
	Message      Message     `json:"message"`
	Logprobs     interface{} `json:"logprobs"` // This can be null or a complex object
	FinishReason string      `json:"finish_reason"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type OpenAIProvider struct {
	options    OpenAIOptions
	httpClient *http.Client
}

// OpenAIOptions configures an OpenAIProvider. Sampling parameters are left to
// the server defaults: reasoning models reject max_tokens and custom
// temperatures, and a reply is a single command anyway.
type OpenAIOptions struct {
	*ProviderOptions
	model       string
	messages    []map[string]string
	timeout     time.Duration
	withContext bool
}

// osName returns a human friendly name for the operating system aiterm runs on,
// so the model can tailor commands (e.g. BSD vs GNU flags on macOS).
func osName(goos string) string {
	switch goos {
	case "darwin":
		return "macOS"
	case "linux":
		return "Linux"
	case "freebsd":
		return "FreeBSD"
	default:
		return goos
	}
}

func systemPrompt(goos string) string {
	return "You are a command line interpreter that converts the user's natural language request " +
		"into the closest and most accurate Unix shell command for " + osName(goos) + ". " +
		"Output only the command, with no instructions, explanations, markdown or code fences. " +
		"If the request does not resemble a command, reply exactly: not a command"
}

func defaultMessages() []map[string]string {
	return []map[string]string{
		{
			"role":    "system",
			"content": systemPrompt(runtime.GOOS),
		},
	}
}

func NewOpenAIProvider(apiKey string, options *OpenAIOptions) APIProvider {
	if options == nil {
		options = &OpenAIOptions{
			ProviderOptions: &ProviderOptions{
				URL:    defaultEndPoint,
				APIKey: apiKey,
			},
			messages: defaultMessages(),
			model:    defaultModel,
		}
	}
	if options.ProviderOptions == nil {
		options.ProviderOptions = &ProviderOptions{URL: defaultEndPoint,
			APIKey: apiKey,
		}
	} else {
		options.APIKey = apiKey
		if options.URL == "" {
			options.URL = defaultEndPoint
		}
	}
	if options.model == "" {
		options.model = defaultModel
	}
	if options.messages == nil {
		options.messages = defaultMessages()
	}
	if options.timeout <= 0 {
		options.timeout = defaultTimeout
	}
	return &OpenAIProvider{
		options:    *options,
		httpClient: &http.Client{Timeout: options.timeout},
	}
}

func (o *OpenAIProvider) clearMessages() {
	o.options.messages = defaultMessages()
}
func (o *OpenAIProvider) constructPayload(messages []map[string]string) map[string]interface{} {
	return map[string]interface{}{
		"model":    o.options.model,
		"messages": messages,
	}
}

func (o *OpenAIProvider) newFetchConfig(withCtxt bool) FetchConfig {
	return func(o interface{}) {
		if provider, ok := o.(*OpenAIProvider); ok {
			provider.options.withContext = withCtxt
		}
	}
}

func (o *OpenAIProvider) setAPIKey(apikey string) {
	o.options.APIKey = apikey
}
func (o *OpenAIProvider) fetch(userRequest string, opts ...FetchConfig) (string, error) {
	for _, opt := range opts {
		opt(o)
	}
	if !o.options.withContext {
		o.clearMessages()
	}

	// Only commit the exchange to the conversation history once it succeeded,
	// so a failed request does not leave a dangling user message behind.
	userMessage := map[string]string{"role": "user", "content": userRequest}
	messages := make([]map[string]string, 0, len(o.options.messages)+1)
	messages = append(messages, o.options.messages...)
	messages = append(messages, userMessage)

	// Construct the request payload
	payloadBytes, err := json.Marshal(o.constructPayload(messages))
	if err != nil {
		return "", &MarshalingError{err}
	}

	// Make the HTTP POST request
	req, err := http.NewRequest(http.MethodPost, o.options.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", &RequestCreationError{err}
	}

	// Set the necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.options.APIKey)

	// Execute the request
	resp, err := o.httpClient.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			err = fmt.Errorf("no response within %v, raise the limit with -timeout or %s: %w", o.httpClient.Timeout, varTimeoutName, err)
		}
		return "", &ExecutionError{err}
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Printf("could not close the response body: %v", err)
		}
	}()

	responseBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes+1))
	if err != nil {
		return "", &ResponseReadError{err}
	}
	if len(responseBody) > maxResponseBytes {
		return "", &ResponseReadError{fmt.Errorf("response is larger than %d bytes", maxResponseBytes)}
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", parseErrorResponse(resp.StatusCode, responseBody)
	}

	var response SuccessResponse
	if err := json.Unmarshal(responseBody, &response); err != nil {
		return "", &UnMarshalingError{err}
	}
	if len(response.Choices) == 0 {
		return "", errors.New("success response does not contain choices")
	}

	res := response.Choices[0].Message.Content
	messages = append(messages, map[string]string{"role": "assistant", "content": res})
	o.options.messages = messages
	return res, nil
}

// parseErrorResponse turns a non-2xx response into an *OAIAPIError. Bodies that
// are not OpenAI shaped JSON (e.g. an HTML page from a proxy) still produce a
// useful error that carries the HTTP status.
func parseErrorResponse(statusCode int, body []byte) error {
	apiErr := &OAIAPIError{StatusCode: statusCode}

	var errorResponse ErrorResponse
	if err := json.Unmarshal(body, &errorResponse); err == nil && errorResponse.Error.Message != "" {
		apiErr.Type = errorResponse.Error.Type
		apiErr.Message = errorResponse.Error.Message
		apiErr.Code = errorResponse.Error.Code
		return apiErr
	}

	apiErr.Type = "http_error"
	apiErr.Message = http.StatusText(statusCode)
	if snippet := truncate(string(bytes.TrimSpace(body)), 200); snippet != "" {
		apiErr.Message = fmt.Sprintf("%s: %s", apiErr.Message, snippet)
	}
	return apiErr
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

func (o *OpenAIProvider) hasAPIKey() bool {
	return o.options.APIKey != ""
}

func (o *OpenAIProvider) handleAPIError(err error) error {
	var apiErr *OAIAPIError
	if errors.As(err, &apiErr) {
		if apiErr.Code == "invalid_api_key" || apiErr.StatusCode == http.StatusUnauthorized {
			return &APIKeyError{OriginalError: apiErr}
		}
		return apiErr
	}
	return err
}
