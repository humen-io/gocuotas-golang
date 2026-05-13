package clientv1_test

import (
	"github.com/humen-io/gocuotas/go/clientv1"
)

// ExampleNewClient shows constructing the Client V1 with default production base URL.
func ExampleNewClient() {
	cfg := clientv1.DefaultClientV1Config()
	_ = clientv1.NewClientWithConfig(cfg)
	// With GOCUOTAS_COMMERCE_API_KEY set: NewClient().GetCommerceFromEnv(context.Background())
}
