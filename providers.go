package main

type FetchConfig func(o interface{})
type APIProvider interface {
	fetch(userRequest string, opts ...FetchConfig) (string, error)
	hasAPIKey() bool
	handleAPIError(err error) error
	newFetchConfig(withCtxt bool) FetchConfig
	setAPIKey(apikey string)
}
type ProviderOptions struct {
	URL    string
	APIKey string
}
