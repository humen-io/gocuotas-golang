package clientv1

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/humen-io/gocuotas/go/api"
)

// Environment variable names (aligned with the Java client).
const (
	EnvCommerceAPIKey = "GOCUOTAS_COMMERCE_API_KEY"
	EnvLiquidacionID  = "GOCUOTAS_LIQUIDACION_ID"
)

// CommerceAPIKey resolves the commerce API key when methods without an explicit key are used.
// If nil, the client reads GOCUOTAS_COMMERCE_API_KEY from the environment.
type CommerceAPIKey func() (string, error)

// Client is the Go HTTP client for API Client V1 (/api_client/v1/...).
type Client struct {
	cfg            ClientV1Config
	httpClient     *http.Client
	commerceAPIKey CommerceAPIKey
}

func defaultCommerceKeyFromEnv() (string, error) {
	v := strings.TrimSpace(os.Getenv(EnvCommerceAPIKey))
	if v == "" {
		return "", fmt.Errorf("set environment variable %s (commerce API key)", EnvCommerceAPIKey)
	}
	return v, nil
}

func liquidacionIDFromEnv() (string, error) {
	v := strings.TrimSpace(os.Getenv(EnvLiquidacionID))
	if v == "" {
		return "", fmt.Errorf("set environment variable %s (settlement id)", EnvLiquidacionID)
	}
	return v, nil
}

// NewClient builds a client with DefaultClientV1Config and http.DefaultClient.
// CommerceAPIKey defaults to reading GOCUOTAS_COMMERCE_API_KEY.
func NewClient() *Client {
	return NewClientWithConfig(DefaultClientV1Config())
}

// NewClientWithConfig uses the given config and http.DefaultClient.
func NewClientWithConfig(cfg ClientV1Config) *Client {
	return &Client{
		cfg:            cfg,
		httpClient:     http.DefaultClient,
		commerceAPIKey: defaultCommerceKeyFromEnv,
	}
}

// NewClientForTest allows injecting HTTP client and commerce key resolver (used in unit tests).
func NewClientForTest(cfg ClientV1Config, hc *http.Client, keyFn CommerceAPIKey) *Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	if keyFn == nil {
		keyFn = defaultCommerceKeyFromEnv
	}
	return &Client{cfg: cfg, httpClient: hc, commerceAPIKey: keyFn}
}

func (c *Client) commerceKey() (string, error) {
	return c.commerceAPIKey()
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.cfg.RequestTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.cfg.RequestTimeout)
}

func encodePathSegment(s string) string {
	// Java: URLEncoder.encode(id, UTF_8).replace("+", "%20")
	return strings.ReplaceAll(url.PathEscape(s), "+", "%20")
}

func truncateBody(s string) string {
	if len(s) <= 500 {
		return s
	}
	return s[:500] + "…"
}

func (c *Client) doGET(ctx context.Context, path, accept, commerceAPIKey string) ([]byte, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	u, err := c.cfg.Resolve(path)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", accept)
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(commerceAPIKey))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return body, nil
	}
	return nil, &api.APIError{
		StatusCode:   resp.StatusCode,
		ResponseBody: string(body),
		Message:      fmt.Sprintf("GoCuotas API Client V1 error HTTP %d: %s", resp.StatusCode, truncateBody(string(body))),
	}
}

// GetCommerce calls GET /api_client/v1/client (Accept: application/json).
func (c *Client) GetCommerce(ctx context.Context, commerceAPIKey string) (*CommerceResponse, error) {
	body, err := c.doGET(ctx, PathClient, "application/json", commerceAPIKey)
	if err != nil {
		return nil, err
	}
	var out CommerceResponse
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// GetCommerceFromEnv uses GOCUOTAS_COMMERCE_API_KEY.
func (c *Client) GetCommerceFromEnv(ctx context.Context) (*CommerceResponse, error) {
	k, err := c.commerceKey()
	if err != nil {
		return nil, err
	}
	return c.GetCommerce(ctx, k)
}

// ListSettlements calls GET /api_client/v1/expense_settlements.
func (c *Client) ListSettlements(ctx context.Context, commerceAPIKey string) ([]Liquidacion, error) {
	body, err := c.doGET(ctx, PathExpenseSettlements, "application/json", commerceAPIKey)
	if err != nil {
		return nil, err
	}
	var out []Liquidacion
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListSettlementsFromEnv(ctx context.Context) ([]Liquidacion, error) {
	k, err := c.commerceKey()
	if err != nil {
		return nil, err
	}
	return c.ListSettlements(ctx, k)
}

// GetSettlement calls GET /api_client/v1/expense_settlements/{id}.
func (c *Client) GetSettlement(ctx context.Context, commerceAPIKey, settlementID string) (*SettlementInfo, error) {
	path := PathExpenseSettlements + "/" + encodePathSegment(settlementID)
	body, err := c.doGET(ctx, path, "application/json", commerceAPIKey)
	if err != nil {
		return nil, err
	}
	var out SettlementInfo
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) GetSettlementFromEnv(ctx context.Context, settlementID string) (*SettlementInfo, error) {
	k, err := c.commerceKey()
	if err != nil {
		return nil, err
	}
	return c.GetSettlement(ctx, k, settlementID)
}

// GetSettlementFromEnvIDs uses GOCUOTAS_COMMERCE_API_KEY and GOCUOTAS_LIQUIDACION_ID.
func (c *Client) GetSettlementFromEnvIDs(ctx context.Context) (*SettlementInfo, error) {
	k, err := c.commerceKey()
	if err != nil {
		return nil, err
	}
	id, err := liquidacionIDFromEnv()
	if err != nil {
		return nil, err
	}
	return c.GetSettlement(ctx, k, id)
}

// ListSettlementsPlainText calls GET /api_client/v1/expense_settlements_csvs (Accept: text/plain).
func (c *Client) ListSettlementsPlainText(ctx context.Context, commerceAPIKey string) (string, error) {
	body, err := c.doGET(ctx, PathExpenseSettlementsCSVs, "text/plain", commerceAPIKey)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) ListSettlementsPlainTextFromEnv(ctx context.Context) (string, error) {
	k, err := c.commerceKey()
	if err != nil {
		return "", err
	}
	return c.ListSettlementsPlainText(ctx, k)
}

// GetSettlementPlainText calls GET /api_client/v1/expense_settlements_csvs/{id} (Accept: text/plain).
func (c *Client) GetSettlementPlainText(ctx context.Context, commerceAPIKey, settlementID string) (string, error) {
	path := PathExpenseSettlementsCSVs + "/" + encodePathSegment(settlementID)
	body, err := c.doGET(ctx, path, "text/plain", commerceAPIKey)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func (c *Client) GetSettlementPlainTextFromEnv(ctx context.Context, settlementID string) (string, error) {
	k, err := c.commerceKey()
	if err != nil {
		return "", err
	}
	return c.GetSettlementPlainText(ctx, k, settlementID)
}

func (c *Client) GetSettlementPlainTextFromEnvIDs(ctx context.Context) (string, error) {
	k, err := c.commerceKey()
	if err != nil {
		return "", err
	}
	id, err := liquidacionIDFromEnv()
	if err != nil {
		return "", err
	}
	return c.GetSettlementPlainText(ctx, k, id)
}
