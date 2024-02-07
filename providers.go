package main

type FetchConfig func(o interface{})
type APIProvider interface {
	fetch(userRequest string, opts ...FetchConfig) (string, error)
}
type ProviderOptions struct {
	URL    string
	APIKey string
}
