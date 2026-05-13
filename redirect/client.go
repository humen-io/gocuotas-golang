package redirect

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"

	"github.com/humen-io/gocuotas/go/api"
)

// Environment variable names (aligned with the Java redirect client).
const (
	EnvJWT            = "GOCUOTAS_JWT"
	EnvEmail          = "GOCUOTAS_EMAIL"
	EnvAPIKey         = "GOCUOTAS_API_KEY"
	EnvDeliveredStart = "GOCUOTAS_DELIVERED_START"
	EnvDeliveredEnd   = "GOCUOTAS_DELIVERED_END"
)

// PanelAPIKey resolves GOCUOTAS_API_KEY when nil, NewClient uses os.Getenv.
type PanelAPIKey func() (string, error)

// Client is the HTTP client for API Redirect V1.
type Client struct {
	cfg         Config
	httpClient  *http.Client
	panelAPIKey PanelAPIKey

	mu        sync.Mutex
	cachedJWT string
}

func readPanelAPIKeyFromEnv() (string, error) {
	v := strings.TrimSpace(os.Getenv(EnvAPIKey))
	if v == "" {
		return "", fmt.Errorf("set environment variable %s", EnvAPIKey)
	}
	return v, nil
}

// NewClient uses DefaultConfig, http.DefaultClient, and reads GOCUOTAS_API_KEY from the environment for checkout / auth password.
func NewClient() *Client {
	return NewClientWithConfig(DefaultConfig())
}

func NewClientWithConfig(cfg Config) *Client {
	return &Client{
		cfg:         cfg,
		httpClient:  http.DefaultClient,
		panelAPIKey: readPanelAPIKeyFromEnv,
	}
}

// NewClientForTest injects HTTP client and optional panel API key resolver (nil uses env).
func NewClientForTest(cfg Config, hc *http.Client, panelKey PanelAPIKey) *Client {
	if hc == nil {
		hc = http.DefaultClient
	}
	if panelKey == nil {
		panelKey = readPanelAPIKeyFromEnv
	}
	return &Client{cfg: cfg, httpClient: hc, panelAPIKey: panelKey}
}

func (c *Client) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if c.cfg.RequestTimeout <= 0 {
		return ctx, func() {}
	}
	return context.WithTimeout(ctx, c.cfg.RequestTimeout)
}

func encodePathSegment(s string) string {
	return strings.ReplaceAll(url.PathEscape(s), "+", "%20")
}

func truncateBody(s string) string {
	if len(s) <= 500 {
		return s
	}
	return s[:500] + "…"
}

func (c *Client) panelKey() (string, error) {
	return c.panelAPIKey()
}

// bearerForOrderRequests matches Java: GOCUOTAS_JWT if set, else one-shot authenticate(email, apiKey as password) with cache.
func (c *Client) bearerForOrderRequests(ctx context.Context) (string, error) {
	if v := strings.TrimSpace(os.Getenv(EnvJWT)); v != "" {
		return v, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.cachedJWT != "" {
		return c.cachedJWT, nil
	}
	email := strings.TrimSpace(os.Getenv(EnvEmail))
	if email == "" {
		return "", fmt.Errorf("for order calls without explicit bearer, set %s or %s + %s", EnvJWT, EnvEmail, EnvAPIKey)
	}
	pw, err := c.panelKey()
	if err != nil {
		return "", err
	}
	ar, err := c.Authenticate(ctx, email, pw)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(ar.Token) == "" {
		return "", fmt.Errorf("gocuotas: authenticate response had no token")
	}
	c.cachedJWT = strings.TrimSpace(ar.Token)
	return c.cachedJWT, nil
}

func deliveredRangeFromEnv() (start, end string, err error) {
	start = strings.TrimSpace(os.Getenv(EnvDeliveredStart))
	end = strings.TrimSpace(os.Getenv(EnvDeliveredEnd))
	if start == "" || end == "" {
		return "", "", fmt.Errorf("set %s and %s (format YYYY-MM-DD HH:mm)", EnvDeliveredStart, EnvDeliveredEnd)
	}
	return start, end, nil
}

func (c *Client) ordersListURL(deliveredStart, deliveredEnd string) (*url.URL, error) {
	q := url.Values{}
	q.Set("delivered_start", deliveredStart)
	q.Set("delivered_end", deliveredEnd)
	return c.cfg.Resolve(PathOrders + "?" + q.Encode())
}

// Authenticate POSTs /api_redirect/v1/authentication (no Authorization header).
func (c *Client) Authenticate(ctx context.Context, email, password string) (*AuthenticationResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	u, err := c.cfg.Resolve(PathAuthentication)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(AuthenticationRequest{Email: email, Password: password})
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &api.APIError{
			StatusCode:   resp.StatusCode,
			ResponseBody: string(rb),
			Message:      fmt.Sprintf("GoCuotas API error HTTP %d: %s", resp.StatusCode, truncateBody(string(rb))),
		}
	}
	var out AuthenticationResponse
	if err := json.Unmarshal(rb, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateCheckout POSTs /api_redirect/v1/checkouts with Authorization: Bearer bearerToken (JWT or panel API key per your integration).
func (c *Client) CreateCheckout(ctx context.Context, bearerToken string, checkout CreateCheckoutRequest) (*CreateCheckoutResponse, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	u, err := c.cfg.Resolve(PathCheckouts)
	if err != nil {
		return nil, err
	}
	body, err := json.Marshal(checkout)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u.String(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &api.APIError{
			StatusCode:   resp.StatusCode,
			ResponseBody: string(rb),
			Message:      fmt.Sprintf("GoCuotas API error HTTP %d: %s", resp.StatusCode, truncateBody(string(rb))),
		}
	}
	var out CreateCheckoutResponse
	if err := json.Unmarshal(rb, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CreateCheckoutFromEnv uses GOCUOTAS_API_KEY as Bearer (same as Java createCheckout(CreateCheckoutRequest)).
func (c *Client) CreateCheckoutFromEnv(ctx context.Context, checkout CreateCheckoutRequest) (*CreateCheckoutResponse, error) {
	k, err := c.panelKey()
	if err != nil {
		return nil, err
	}
	return c.CreateCheckout(ctx, k, checkout)
}

// ListOrders GETs /api_redirect/v1/orders with delivered_start and delivered_end query parameters.
func (c *Client) ListOrders(ctx context.Context, bearerToken, deliveredStart, deliveredEnd string) (string, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	u, err := c.ordersListURL(strings.TrimSpace(deliveredStart), strings.TrimSpace(deliveredEnd))
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", &api.APIError{
			StatusCode:   resp.StatusCode,
			ResponseBody: string(rb),
			Message:      fmt.Sprintf("GoCuotas API error HTTP %d: %s", resp.StatusCode, truncateBody(string(rb))),
		}
	}
	return string(rb), nil
}

// ListOrdersFromEnvDelivered uses GOCUOTAS_DELIVERED_START and GOCUOTAS_DELIVERED_END.
func (c *Client) ListOrdersFromEnvDelivered(ctx context.Context, bearerToken string) (string, error) {
	ds, de, err := deliveredRangeFromEnv()
	if err != nil {
		return "", err
	}
	return c.ListOrders(ctx, bearerToken, ds, de)
}

// ListOrdersAuto uses JWT-or-authenticate bearer plus delivered range from environment variables.
func (c *Client) ListOrdersAuto(ctx context.Context) (string, error) {
	b, err := c.bearerForOrderRequests(ctx)
	if err != nil {
		return "", err
	}
	return c.ListOrdersFromEnvDelivered(ctx, b)
}

// ListOrdersJSON validates JSON and returns raw bytes (same payload as ListOrders).
func (c *Client) ListOrdersJSON(ctx context.Context, bearerToken, deliveredStart, deliveredEnd string) (json.RawMessage, error) {
	s, err := c.ListOrders(ctx, bearerToken, deliveredStart, deliveredEnd)
	if err != nil {
		return nil, err
	}
	if !json.Valid([]byte(s)) {
		return nil, fmt.Errorf("gocuotas: list orders response is not valid JSON")
	}
	return json.RawMessage(s), nil
}

func (c *Client) sendOrderByID(logicalOrderID string, req *http.Request) (string, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	rb, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return string(rb), nil
	}
	if resp.StatusCode == HTTPStatusOrderNotFound {
		return "", &OrderNotFoundError{OrderID: logicalOrderID, ResponseBody: string(rb)}
	}
	return "", &api.APIError{
		StatusCode:   resp.StatusCode,
		ResponseBody: string(rb),
		Message:      fmt.Sprintf("GoCuotas API error HTTP %d: %s", resp.StatusCode, truncateBody(string(rb))),
	}
}

// GetOrder GETs /api_redirect/v1/orders/{id}.
func (c *Client) GetOrder(ctx context.Context, bearerToken, orderID string) (string, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	path := PathOrders + "/" + encodePathSegment(orderID)
	u, err := c.cfg.Resolve(path)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
	return c.sendOrderByID(orderID, req)
}

// GetOrderAuto uses JWT-or-authenticate bearer.
func (c *Client) GetOrderAuto(ctx context.Context, orderID string) (string, error) {
	b, err := c.bearerForOrderRequests(ctx)
	if err != nil {
		return "", err
	}
	return c.GetOrder(ctx, b, orderID)
}

// GetOrderJSON is GetOrder with JSON validation (Java buscarOrden JsonNode).
func (c *Client) GetOrderJSON(ctx context.Context, bearerToken, orderID string) (json.RawMessage, error) {
	s, err := c.GetOrder(ctx, bearerToken, orderID)
	if err != nil {
		return nil, err
	}
	if !json.Valid([]byte(s)) {
		return nil, fmt.Errorf("gocuotas: get order response is not valid JSON")
	}
	return json.RawMessage(s), nil
}

// RefundOrder DELETEs /api_redirect/v1/orders/{id} with JSON Accept/Content-Type headers.
func (c *Client) RefundOrder(ctx context.Context, bearerToken, orderID string) (string, error) {
	ctx, cancel := c.withTimeout(ctx)
	defer cancel()
	path := PathOrders + "/" + encodePathSegment(orderID)
	u, err := c.cfg.Resolve(path)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, u.String(), nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(bearerToken))
	return c.sendOrderByID(orderID, req)
}

// RefundOrderAuto uses JWT-or-authenticate bearer.
func (c *Client) RefundOrderAuto(ctx context.Context, orderID string) (string, error) {
	b, err := c.bearerForOrderRequests(ctx)
	if err != nil {
		return "", err
	}
	return c.RefundOrder(ctx, b, orderID)
}

// RefundOrderJSON validates JSON (Java reembolsarOrden).
func (c *Client) RefundOrderJSON(ctx context.Context, bearerToken, orderID string) (json.RawMessage, error) {
	s, err := c.RefundOrder(ctx, bearerToken, orderID)
	if err != nil {
		return nil, err
	}
	if !json.Valid([]byte(s)) {
		return nil, fmt.Errorf("gocuotas: refund response is not valid JSON")
	}
	return json.RawMessage(s), nil
}

// GetOrderJSONAuto uses JWT-or-authenticate bearer (same idea as Java buscarOrden() without explicit token).
func (c *Client) GetOrderJSONAuto(ctx context.Context, orderID string) (json.RawMessage, error) {
	b, err := c.bearerForOrderRequests(ctx)
	if err != nil {
		return nil, err
	}
	return c.GetOrderJSON(ctx, b, orderID)
}
