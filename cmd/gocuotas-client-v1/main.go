// Command gocuotas-client-v1 calls Commerce API Client V1 using environment variables.
//
// Usage:
//
//	GOCUOTAS_COMMERCE_API_KEY=... go run ./cmd/gocuotas-client-v1/ commerce
//	GOCUOTAS_COMMERCE_API_KEY=... go run ./cmd/gocuotas-client-v1/ settlements
//	GOCUOTAS_COMMERCE_API_KEY=... GOCUOTAS_LIQUIDACION_ID=9001001 go run ./cmd/gocuotas-client-v1/ settlement
//	GOCUOTAS_COMMERCE_API_KEY=... go run ./cmd/gocuotas-client-v1/ settlements-csv
//	GOCUOTAS_COMMERCE_API_KEY=... GOCUOTAS_LIQUIDACION_ID=9001001 go run ./cmd/gocuotas-client-v1/ settlement-csv
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/humen-io/gocuotas/go/clientv1"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: gocuotas-client-v1 <commerce|settlements|settlement|settlements-csv|settlement-csv>")
		os.Exit(2)
	}
	ctx := context.Background()
	c := clientv1.NewClient()
	switch os.Args[1] {
	case "commerce":
		info, err := c.GetCommerceFromEnv(ctx)
		if err != nil {
			exitErr(err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(info)
	case "settlements":
		list, err := c.ListSettlementsFromEnv(ctx)
		if err != nil {
			exitErr(err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(list)
	case "settlement":
		info, err := c.GetSettlementFromEnvIDs(ctx)
		if err != nil {
			exitErr(err)
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(info)
	case "settlements-csv":
		s, err := c.ListSettlementsPlainTextFromEnv(ctx)
		if err != nil {
			exitErr(err)
		}
		fmt.Print(s)
	case "settlement-csv":
		s, err := c.GetSettlementPlainTextFromEnvIDs(ctx)
		if err != nil {
			exitErr(err)
		}
		fmt.Print(s)
	default:
		fmt.Fprintln(os.Stderr, "unknown subcommand")
		os.Exit(2)
	}
}

func exitErr(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
