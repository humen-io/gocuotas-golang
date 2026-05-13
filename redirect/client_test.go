package redirect

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/humen-io/gocuotas/go/api"
)

func TestGetOrder_usesGetWithEncodedID(t *testing.T) {
	var method, rawPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		rawPath = r.URL.EscapedPath()
		if !strings.HasPrefix(rawPath, PathOrders) {
			t.Fatalf("path %s", rawPath)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"x"}`))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "x", nil })

	body, err := c.GetOrder(context.Background(), "tok", "a/b")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodGet {
		t.Fatalf("method %s", method)
	}
	if !strings.HasSuffix(rawPath, "/api_redirect/v1/orders/a%2Fb") {
		t.Fatalf("escaped path %q", rawPath)
	}
	if !strings.Contains(body, `"id":"x"`) {
		t.Fatalf("body %q", body)
	}
}

func TestAuthenticate_postsJSONWithoutAuthHeader(t *testing.T) {
	var captured string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathAuthentication || r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Authorization") != "" {
			t.Fatal("unexpected Authorization on authenticate")
		}
		b, _ := io.ReadAll(r.Body)
		captured = string(b)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"token":"jwt-from-server"}`))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "unused", nil })

	ar, err := c.Authenticate(context.Background(), "seller@example.com", "secret")
	if err != nil {
		t.Fatal(err)
	}
	if ar.Token != "jwt-from-server" {
		t.Fatalf("token %q", ar.Token)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(captured), &m); err != nil {
		t.Fatal(err)
	}
	if m["email"] != "seller@example.com" || m["password"] != "secret" {
		t.Fatalf("payload %v", m)
	}
}

func TestAuthenticationResponse_unmarshalAccessTokenAlias(t *testing.T) {
	var ar AuthenticationResponse
	if err := json.Unmarshal([]byte(`{"access_token":"from-alias"}`), &ar); err != nil {
		t.Fatal(err)
	}
	if ar.Token != "from-alias" {
		t.Fatalf("got %q", ar.Token)
	}
}

func TestCreateCheckout_sendsBearerAndReturnsURLInit(t *testing.T) {
	var auth string
	var body []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathCheckouts {
			t.Fatalf("path %s", r.URL.Path)
		}
		auth = r.Header.Get("Authorization")
		body, _ = io.ReadAll(r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"url_init":"https://pay.gocuotas.example/start"}`))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "unused", nil })

	req := NewCheckoutBuilder().
		AmountInCents(10_000).
		Email("buyer@example.com").
		OrderReferenceID("ORD-1").
		PhoneNumber("1144440000").
		URLSuccess("https://shop.example/ok").
		URLFailure("https://shop.example/ko").
		Build()

	res, err := c.CreateCheckout(context.Background(), "api-key-or-jwt", req)
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer api-key-or-jwt" {
		t.Fatalf("auth %q", auth)
	}
	if res.URLInit != "https://pay.gocuotas.example/start" {
		t.Fatalf("url %q", res.URLInit)
	}
	var tree map[string]any
	if err := json.Unmarshal(body, &tree); err != nil {
		t.Fatal(err)
	}
	if int64(tree["amount_in_cents"].(float64)) != 10_000 {
		t.Fatalf("amount %v", tree["amount_in_cents"])
	}
	if tree["order_reference_id"] != "ORD-1" {
		t.Fatalf("ref %v", tree["order_reference_id"])
	}
	_, has := tree["webhook_url"]
	if has && tree["webhook_url"] != nil {
		t.Fatalf("webhook should be absent or null, got %v", tree["webhook_url"])
	}
}

func TestCreateCheckoutFromEnv_usesPanelKeySupplier(t *testing.T) {
	var auth string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"url_init":"https://pay.example/u"}`))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "key-from-GOCUOTAS_API_KEY", nil })

	req := NewCheckoutBuilder().
		AmountInCents(1).
		Email("e@e.e").
		OrderReferenceID("r").
		PhoneNumber("1").
		URLSuccess("https://a").
		URLFailure("https://b").
		Build()

	res, err := c.CreateCheckoutFromEnv(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer key-from-GOCUOTAS_API_KEY" {
		t.Fatalf("auth %q", auth)
	}
	if res.URLInit != "https://pay.example/u" {
		t.Fatalf("url %q", res.URLInit)
	}
}

func TestCreateCheckout_HTTP422(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte(`{"error":"invalid_amount"}`))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "k", nil })

	req := NewCheckoutBuilder().AmountInCents(1).Email("a@b.c").OrderReferenceID("x").PhoneNumber("1").URLSuccess("https://a").URLFailure("https://b").Build()
	_, err := c.CreateCheckout(context.Background(), "t", req)
	if err == nil {
		t.Fatal("expected error")
	}
	var ae *api.APIError
	if !errors.As(err, &ae) || ae.StatusCode != 422 {
		t.Fatalf("got %v", err)
	}
	if !strings.Contains(ae.ResponseBody, "invalid_amount") {
		t.Fatalf("body %q", ae.ResponseBody)
	}
}

func TestRefundOrder_usesDeleteWithJSONHeaders(t *testing.T) {
	var method, ct string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == PathOrders+"/ref-42" {
			method = r.Method
			ct = r.Header.Get("Content-Type")
			if r.Method == http.MethodDelete {
				w.Header().Set("Content-Type", "application/json")
				_, _ = w.Write([]byte(`[{"id":7,"amount_in_cents":100,"status":"refunded"}]`))
				return
			}
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "x", nil })

	body, err := c.RefundOrder(context.Background(), "tok", "ref-42")
	if err != nil {
		t.Fatal(err)
	}
	if method != http.MethodDelete || ct != "application/json" {
		t.Fatalf("method=%s ct=%s", method, ct)
	}
	if !strings.Contains(body, `"id":7`) {
		t.Fatalf("body %q", body)
	}
}

func TestRefundOrderJSON_parsesArray(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == PathOrders+"/x-1" && r.Method == http.MethodDelete {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"id":1,"amount_in_cents":50,"status":"ok"}]`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "x", nil })

	raw, err := c.RefundOrderJSON(context.Background(), "tok", "x-1")
	if err != nil {
		t.Fatal(err)
	}
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err != nil {
		t.Fatal(err)
	}
	if len(arr) != 1 || int(arr[0]["id"].(float64)) != 1 || arr[0]["status"] != "ok" {
		t.Fatalf("%s", string(raw))
	}
}

func TestGetOrder_HTTP404_OrderNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == PathOrders+"/absent-99" && r.Method == http.MethodGet {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte(`{"message":"not found"}`))
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "x", nil })

	_, err := c.GetOrder(context.Background(), "tok", "absent-99")
	if err == nil {
		t.Fatal("expected error")
	}
	var onf *OrderNotFoundError
	if !errors.As(err, &onf) || onf.OrderID != "absent-99" {
		t.Fatalf("got %T %v", err, err)
	}
	if !strings.Contains(onf.ResponseBody, "not found") {
		t.Fatalf("body %q", onf.ResponseBody)
	}
}

func TestGetOrderJSON_HTTP404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "x", nil })

	_, err := c.GetOrderJSON(context.Background(), "tok", "missing-ref")
	var onf *OrderNotFoundError
	if !errors.As(err, &onf) || onf.OrderID != "missing-ref" {
		t.Fatalf("got %v", err)
	}
}

func TestRefundOrder_HTTP404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusMethodNotAllowed)
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "x", nil })

	_, err := c.RefundOrder(context.Background(), "tok", "gone-7")
	var onf *OrderNotFoundError
	if !errors.As(err, &onf) || onf.OrderID != "gone-7" {
		t.Fatalf("got %v", err)
	}
}

func TestListOrders_HTTP404_isAPIErrorNotOrderNotFound(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == PathOrders {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "x", nil })

	_, err := c.ListOrders(context.Background(), "tok", "2020-01-01 00:00", "2020-02-01 00:00")
	var ae *api.APIError
	if !errors.As(err, &ae) || ae.StatusCode != 404 {
		t.Fatalf("got %v", err)
	}
	var onf *OrderNotFoundError
	if errors.As(err, &onf) {
		t.Fatal("list 404 must not be OrderNotFoundError")
	}
}

func TestListOrders_queryParams(t *testing.T) {
	var method, rawQuery string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		rawQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "x", nil })

	s, err := c.ListOrders(context.Background(), "tok", "2021-01-19 10:00", "2024-12-19 20:00")
	if err != nil || s != "[]" {
		t.Fatalf("err=%v s=%q", err, s)
	}
	if method != http.MethodGet {
		t.Fatalf("method %s", method)
	}
	if !strings.Contains(rawQuery, "delivered_start=") || !strings.Contains(rawQuery, "delivered_end=") {
		t.Fatalf("query %q", rawQuery)
	}
}

func TestListOrdersJSON_array(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"order_reference_id":"A"}]`))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "x", nil })

	raw, err := c.ListOrdersJSON(context.Background(), "tok", "2021-01-19 10:00", "2024-12-19 20:00")
	if err != nil {
		t.Fatal(err)
	}
	var arr []map[string]any
	if err := json.Unmarshal(raw, &arr); err != nil || len(arr) != 1 || arr[0]["order_reference_id"] != "A" {
		t.Fatalf("%s err=%v", string(raw), err)
	}
}

func TestGetOrderJSON_object(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"order_reference_id":"PED-1","status":"approved"}`))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "x", nil })

	raw, err := c.GetOrderJSON(context.Background(), "tok", "PED-1")
	if err != nil {
		t.Fatal(err)
	}
	var obj map[string]any
	if err := json.Unmarshal(raw, &obj); err != nil {
		t.Fatal(err)
	}
	if obj["order_reference_id"] != "PED-1" || obj["status"] != "approved" {
		t.Fatalf("%v", obj)
	}
}
