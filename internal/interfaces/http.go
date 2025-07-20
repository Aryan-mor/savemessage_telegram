package interfaces

import "net/http"

// HTTPClient defines the interface for an HTTP client.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	Get(url string) (*http.Response, error)
}
