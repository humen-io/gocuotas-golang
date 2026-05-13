package clientv1

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/humen-io/gocuotas/go/api"
)

func TestGetCommerce_sendsBearerAcceptJSON(t *testing.T) {
	var auth, accept string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth = r.Header.Get("Authorization")
		accept = r.Header.Get("Accept")
		if r.URL.Path != PathClient {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":1000001,"name":"Comercio de ejemplo S.R.L.","cuit":"20987654321","surcharge_percentage_to_online_orders":"0.0","max_number_of_installments":3}`))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := ClientV1Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "api-key-x", nil })

	info, err := c.GetCommerce(context.Background(), "api-key-x")
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer api-key-x" {
		t.Fatalf("Authorization: %q", auth)
	}
	if accept != "application/json" {
		t.Fatalf("Accept: %q", accept)
	}
	if info.ID != 1000001 || info.Name != "Comercio de ejemplo S.R.L." || info.Cuit != "20987654321" {
		t.Fatalf("%+v", info)
	}
	if info.SurchargePercentageToOnline != "0.0" || info.MaxNumberOfInstallments != 3 {
		t.Fatalf("%+v", info)
	}
}

func TestListSettlements_parsesArray(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != PathExpenseSettlements {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		payload := `[{"id":9001001,"payment_expense_method":"transferencia","payment_expense_at":"2026-05-12","due_expense_at":"2026-05-12","payment_expense_retained_amount_in_cents":0,"payment_expense_amount_in_cents":2274953}]`
		_, _ = w.Write([]byte(payload))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := ClientV1Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "k1", nil })

	list, err := c.ListSettlements(context.Background(), "k1")
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("len=%d", len(list))
	}
	l := list[0]
	if l.ID != 9001001 || l.PaymentExpenseMethod != "transferencia" {
		t.Fatalf("%+v", l)
	}
	if l.PaymentExpenseAmountInCents != 2274953 {
		t.Fatalf("cents %d", l.PaymentExpenseAmountInCents)
	}
}

func TestGetSettlement_parsesDetail(t *testing.T) {
	jsonBody := `{"id":9001001,"payment_expense_method":"transferencia","payment_expense_at":"2026-05-12","due_expense_at":"2026-05-12","payment_expense_retained_amount_in_cents":0,"payment_expense_amount_in_cents":2274953,"details":[{"id":99,"description":"Pedido","delivered_at":"2026-05-01","due_expense_at":"2026-05-12","amount_in_cents":100,"commission_amount_in_cents":1,"tax_amount_in_cents":2,"expense_amount_in_cents":3,"discarded_at":null,"status":"approved","number_of_installments":3,"order_reference_id":"ORD-1","payment":{"card":{"number":"406651******6008","name":"Visa"}}}],"normal_retention_retain_paid_orders":[{}],"tax_retention_retain_paid_orders":[]}`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		want := PathExpenseSettlements + "/9001001"
		if r.URL.Path != want {
			t.Fatalf("path %q want %q", r.URL.Path, want)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(jsonBody))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := ClientV1Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "k1", nil })

	info, err := c.GetSettlement(context.Background(), "k1", "9001001")
	if err != nil {
		t.Fatal(err)
	}
	if info.ID != 9001001 || len(info.Details) != 1 {
		t.Fatalf("%+v", info)
	}
	d := info.Details[0]
	if d.Payment == nil || d.Payment.Card == nil || d.Payment.Card.Number != "406651******6008" {
		t.Fatalf("%+v", d.Payment)
	}
	if len(info.NormalRetentionRetainPaidOrders) != 1 || len(info.TaxRetentionRetainPaidOrders) != 0 {
		t.Fatalf("retention lens normal=%d tax=%d", len(info.NormalRetentionRetainPaidOrders), len(info.TaxRetentionRetainPaidOrders))
	}
}

func TestListSettlementsPlainText_acceptTextPlain(t *testing.T) {
	var accept string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		accept = r.Header.Get("Accept")
		if r.URL.Path != PathExpenseSettlementsCSVs {
			t.Fatalf("path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		csv := "ID,Método de Pago,Fecha de Pago,Fecha de Vencimiento,Monto Retenido,Monto Total\n9001001,transferencia,12/05/2026,12/05/2026,0.0,22749.53\n"
		_, _ = w.Write([]byte(csv))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := ClientV1Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "k1", nil })

	body, err := c.ListSettlementsPlainText(context.Background(), "k1")
	if err != nil {
		t.Fatal(err)
	}
	if accept != "text/plain" {
		t.Fatalf("Accept %q", accept)
	}
	if !strings.Contains(body, "9001001,transferencia") || !strings.Contains(body, "22749.53") {
		t.Fatalf("body %q", body)
	}
}

func TestGetSettlementPlainText_pathAndAccept(t *testing.T) {
	var pathSeen, accept string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		pathSeen = r.URL.Path
		accept = r.Header.Get("Accept")
		csv := "Descripcion,Fecha Origen,Fecha Pago\nVentas,30/01/2026,12/05/2026\nTotales,\"\",\"\"\n"
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte(csv))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := ClientV1Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "k1", nil })

	body, err := c.GetSettlementPlainText(context.Background(), "k1", "9001001")
	if err != nil {
		t.Fatal(err)
	}
	wantSuffix := PathExpenseSettlementsCSVs + "/9001001"
	if !strings.HasSuffix(pathSeen, wantSuffix) {
		t.Fatalf("path %q", pathSeen)
	}
	if accept != "text/plain" {
		t.Fatalf("Accept %q", accept)
	}
	if !strings.Contains(body, "Descripcion,Fecha Origen") {
		t.Fatalf("body %q", body)
	}
}

func TestAPIError_onHTTPError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"bad"}`))
	}))
	defer ts.Close()

	u, _ := url.Parse(ts.URL)
	cfg := ClientV1Config{BaseURL: u, RequestTimeout: 5 * time.Second}
	c := NewClientForTest(cfg, ts.Client(), func() (string, error) { return "k", nil })

	_, err := c.GetCommerce(context.Background(), "k")
	if err == nil {
		t.Fatal("expected error")
	}
	var ae *api.APIError
	if !errors.As(err, &ae) {
		t.Fatalf("want *api.APIError, got %T: %v", err, err)
	}
	if ae.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status %d", ae.StatusCode)
	}
}
