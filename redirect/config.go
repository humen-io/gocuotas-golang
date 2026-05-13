package redirect

import (
	"net/url"
	"time"
)

const (
	DefaultBaseURL = "https://www.gocuotas.com"

	PathAuthentication = "/api_redirect/v1/authentication"
	PathCheckouts      = "/api_redirect/v1/checkouts"
	PathOrders         = "/api_redirect/v1/orders"
)

// Config holds base URL and per-request timeout (GoCuotasRedirectConfig in Java).
type Config struct {
	BaseURL        *url.URL
	RequestTimeout time.Duration
}

func DefaultConfig() Config {
	u, err := url.Parse(DefaultBaseURL)
	if err != nil {
		panic(err)
	}
	return Config{
		BaseURL:        u,
		RequestTimeout: 30 * time.Second,
	}
}

func (c Config) Resolve(path string) (*url.URL, error) {
	if len(path) == 0 || path[0] != '/' {
		path = "/" + path
	}
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	return c.BaseURL.ResolveReference(rel), nil
}
