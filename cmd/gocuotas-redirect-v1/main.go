// Command gocuotas-redirect-v1 exercises Redirect API V1 (authenticate, checkout, orders).
//
// Examples:
//
//	GOCUOTAS_EMAIL=... GOCUOTAS_API_KEY=... go run ./cmd/gocuotas-redirect-v1/ authenticate
//	go run ./cmd/gocuotas-redirect-v1/ checkout
//	GOCUOTAS_JWT=... GOCUOTAS_DELIVERED_START="2021-01-19 10:00" GOCUOTAS_DELIVERED_END="2024-12-19 20:00" go run ./cmd/gocuotas-redirect-v1/ list-orders
//	GOCUOTAS_JWT=... go run ./cmd/gocuotas-redirect-v1/ get-order 80001001
//	GOCUOTAS_EMAIL=... GOCUOTAS_API_KEY=... go run ./cmd/gocuotas-redirect-v1/ get-order 80001001
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/humen-io/gocuotas/go/redirect"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gocuotas-redirect-v1 <authenticate|checkout|list-orders|get-order>")
		os.Exit(2)
	}
	ctx := context.Background()
	c := redirect.NewClient()
	switch os.Args[1] {
	case "authenticate":
		email := os.Getenv("GOCUOTAS_EMAIL")
		pw := os.Getenv("GOCUOTAS_API_KEY")
		if email == "" || pw == "" {
			exitErr(fmt.Errorf("set GOCUOTAS_EMAIL and GOCUOTAS_API_KEY (password)"))
		}
		ar, err := c.Authenticate(ctx, email, pw)
		if err != nil {
			exitErr(err)
		}
		fmt.Println(ar.Token)
	case "checkout":
		req := redirect.NewCheckoutBuilder().
			AmountInCents(150_000).
			Email("comprador@example.com").
			OrderReferenceID("PEDIDO-123").
			PhoneNumber("1144440000").
			URLSuccess("https://mitienda.example/pago/ok").
			URLFailure("https://mitienda.example/pago/error").
			Build()
		res, err := c.CreateCheckoutFromEnv(ctx, req)
		if err != nil {
			exitErr(err)
		}
		fmt.Println(res.URLInit)
	case "list-orders":
		s, err := c.ListOrdersAuto(ctx)
		if err != nil {
			exitErr(err)
		}
		var pretty json.RawMessage = json.RawMessage(s)
		out, err := json.MarshalIndent(pretty, "", "  ")
		if err != nil {
			exitErr(err)
		}
		fmt.Println(string(out))
	case "get-order":
		if len(os.Args) < 3 {
			exitErr(fmt.Errorf("get-order requires order id"))
		}
		var raw json.RawMessage
		var err error
		if jwt := os.Getenv("GOCUOTAS_JWT"); jwt != "" {
			raw, err = c.GetOrderJSON(ctx, jwt, os.Args[2])
		} else {
			raw, err = c.GetOrderJSONAuto(ctx, os.Args[2])
		}
		if err != nil {
			exitErr(err)
		}
		out, err := json.MarshalIndent(raw, "", "  ")
		if err != nil {
			exitErr(err)
		}
		fmt.Println(string(out))
	default:
		fmt.Fprintln(os.Stderr, "unknown subcommand")
		os.Exit(2)
	}
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
