package main

type APIProvider interface {
	fetch(apiKey string, userRequest string) (string, error)
}
type ProviderOptions struct {
	URL string
}
