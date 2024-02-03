package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func api(apiKey string, url string, userRequest string) (string, error) {

	// Construct the request payload
	payload := map[string]interface{}{
		"model": "gpt-3.5-turbo",
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "you are a linux command interpreter that convert the users natural language request to the closest and most accurate unix (linux) commands list only nothing extra. if the requested command does not resemble to a command simply say not a command. ouput only the command everytime no instructions nor explanations.",
			},
			{
				"role":    "user",
				"content": userRequest,
			},
		},
		"temperature":       1.0,
		"max_tokens":        256,
		"top_p":             1.0,
		"frequency_penalty": 0.0,
		"presence_penalty":  0.0,
	}

	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return "", &MarshalingError{err} //fmt.Errorf("failed marshaling payload: %w", err)
	}

	// Make the HTTP POST request
	req, err := http.NewRequest("POST", url, bytes.NewReader(payloadBytes))
	if err != nil {
		return "", &RequestCreationError{err} //fmt.Errorf("failed creating request: %w", err)
	}

	// Set the necessary headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

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
		if len(response.Choices) > 0 {
			return fmt.Sprintf("%s", response.Choices[0].Message.Content), nil
		}
		return "", fmt.Errorf("success response does not contain choices")
	} else {

		var errorResponse ErrorResponse
		err = json.Unmarshal(responseBody, &errorResponse)
		if err != nil {
			return "", &UnMarshalingError{err} //fmt.Errorf("failed unmarshelling the error response: %w", err)
		}
		return "", &APIError{
			Type:    errorResponse.Error.Type,
			Message: errorResponse.Error.Message,
			Code:    errorResponse.Error.Code,
		}
	}

}
