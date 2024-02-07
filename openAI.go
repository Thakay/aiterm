package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
)

const (
	defaultEndPoint = "https://api.openai.com/v1/chat/completions"
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
	options OpenAIOptions
}

type OpenAIOptions struct {
	*ProviderOptions
	model            string
	temp             string
	messages         []map[string]string
	temperature      float64
	maxTokens        int
	topP             float64
	frequencyPenalty float64
	presencePenalty  float64
	withContext      bool
}

func defaultMessages() []map[string]string {
	return []map[string]string{
		{
			"role":    "system",
			"content": "you are a linux command interpreter that convert the users natural language request to the closest and most accurate unix (linux) commands list only nothing extra. if the requested command does not resemble to a command simply say not a command. ouput only the command everytime no instructions nor explanations.",
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
			messages:         defaultMessages(),
			model:            "gpt-3.5-turbo",
			temperature:      1.0,
			maxTokens:        256,
			topP:             1.0,
			frequencyPenalty: 0.0,
			presencePenalty:  0.0,
		}
	}
	if options.ProviderOptions == nil {
		options.ProviderOptions = &ProviderOptions{URL: defaultEndPoint,
			APIKey: apiKey,
		}
	} else {
		options.ProviderOptions.APIKey = apiKey
		if options.ProviderOptions.URL == "" {
			options.ProviderOptions.URL = defaultEndPoint
		}
	}
	if options.messages == nil {
		options.messages = defaultMessages()
	}
	return &OpenAIProvider{*options}
}

func (o *OpenAIProvider) addMessage(role string, message string) {
	o.options.messages = append(o.options.messages, map[string]string{"role": role, "content": message})
}
func (o *OpenAIProvider) clearMessages() {
	o.options.messages = defaultMessages()
}
func (o *OpenAIProvider) constructPayload() map[string]interface{} {
	return map[string]interface{}{
		"model":             o.options.model,
		"messages":          o.options.messages,
		"temperature":       o.options.temperature,
		"max_tokens":        o.options.maxTokens,
		"top_p":             o.options.topP,
		"frequency_penalty": o.options.frequencyPenalty,
		"presence_penalty":  o.options.presencePenalty,
	}
}

func (o *OpenAIProvider) newFetchConfig(withCtxt bool) FetchConfig {
	return func(o interface{}) {
		if provider, ok := o.(*OpenAIProvider); ok {
			provider.options.withContext = withCtxt
		} else {
			log.Fatal("incorrect fetch config provided!")
		}
	}
}

func (o *OpenAIProvider) setAPIKey(apikey string) {
	o.options.APIKey = apikey
}
func (o *OpenAIProvider) fetch(userRequest string, opts ...FetchConfig) (string, error) {
	// Construct the request payload
	for _, opt := range opts {
		opt(o)
	}
	if !o.options.withContext {
		o.clearMessages()
	}
	o.addMessage("user", userRequest)
	payload := o.constructPayload()

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", &MarshalingError{err} //fmt.Errorf("failed marshaling payload: %w", err)
	}

	// Make the HTTP POST request
	req, err := http.NewRequest("POST", o.options.URL, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", &RequestCreationError{err} //fmt.Errorf("failed creating request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.options.APIKey)

	// Execute the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", &ExecutionError{err} //fmt.Errorf("failed executing request: %w", err)
	}
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Fatal("Could not close the Body with the error: ", err)
		}
	}(resp.Body)

	// Read and print the response body
	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", &ResponseReadError{err} //fmt.Errorf("failed reading response body: %w", err)
	}

	if resp.StatusCode >= http.StatusOK && resp.StatusCode <= http.StatusMultipleChoices {

		var response SuccessResponse
		err = json.Unmarshal(responseBody, &response)
		if err != nil {
			return "", &UnMarshalingError{err} //fmt.Errorf("failed unmarshelling the success response: %w", err)
		}
		fmt.Println("-----resp------")
		fmt.Println(response)
		if len(response.Choices) > 0 {
			res := fmt.Sprintf("%s", response.Choices[0].Message.Content)
			o.addMessage("assistant", res)
			return res, nil
		}
		return "", fmt.Errorf("success response does not contain choices")
	} else {

		var errorResponse ErrorResponse
		err = json.Unmarshal(responseBody, &errorResponse)
		if err != nil {
			return "", &UnMarshalingError{err} //fmt.Errorf("failed unmarshelling the error response: %w", err)
		}
		return "", &OAIAPIError{
			Type:    errorResponse.Error.Type,
			Message: errorResponse.Error.Message,
			Code:    errorResponse.Error.Code,
		}
	}
}

func (o *OpenAIProvider) hasAPIKey() bool {
	if o.options.APIKey != "" {
		return true
	}
	return true
}

func (o *OpenAIProvider) handleAPIError(err error) error {
	var apiErr *OAIAPIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case "invalid_api_key":
			return &APIKeyError{OriginalError: apiErr}
		default:
			log.Printf("API error: %v\n", apiErr)
			return apiErr
		}
	}
	return err

}
