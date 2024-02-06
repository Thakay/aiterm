package main

type APIProvider interface {
	fetch(apiKey string, userRequest string) (string, error)
	//modifyOptions(options interface{}) error
}
type ContextualAPIProvider interface {
	APIProvider
	fetchWithContext(apiKey string, userRequest string) (string, error)
}
type ProviderOptions struct {
	URL    string
	APIKey string
}
