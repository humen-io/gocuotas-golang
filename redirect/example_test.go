package redirect_test

import (
	"context"

	"github.com/humen-io/gocuotas/go/redirect"
)

func ExampleNewClient() {
	_ = redirect.NewClient()
	_ = context.Background()
}
