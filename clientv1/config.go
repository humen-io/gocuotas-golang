package clientv1

import (
	"net/url"
	"time"
)

const (
	DefaultBaseURL = "https://www.gocuotas.com"

	PathClient                 = "/api_client/v1/client"
	PathExpenseSettlements     = "/api_client/v1/expense_settlements"
	PathExpenseSettlementsCSVs = "/api_client/v1/expense_settlements_csvs"
)

// ClientV1Config is the Go counterpart of GoCuotasClientV1Config (base URL + per-request timeout budget).
type ClientV1Config struct {
	BaseURL        *url.URL
	RequestTimeout time.Duration
}

func DefaultClientV1Config() ClientV1Config {
	u, err := url.Parse(DefaultBaseURL)
	if err != nil {
		panic(err)
	}
	return ClientV1Config{
		BaseURL:        u,
		RequestTimeout: 60 * time.Second,
	}
}

// Resolve joins an absolute path (e.g. "/api_client/v1/client") with the configured base URL.
func (c ClientV1Config) Resolve(path string) (*url.URL, error) {
	if len(path) == 0 || path[0] != '/' {
		path = "/" + path
	}
	rel, err := url.Parse(path)
	if err != nil {
		return nil, err
	}
	return c.BaseURL.ResolveReference(rel), nil
}
