package redirect

import (
	"fmt"

	"github.com/humen-io/gocuotas/go/api"
)

// HTTPStatusOrderNotFound is the status used for missing orders on GET/DELETE /orders/{id} (Java OrderNotFoundException.HTTP_STATUS).
const HTTPStatusOrderNotFound = 404

// OrderNotFoundError is returned when GET or DELETE /api_redirect/v1/orders/{id} responds 404.
type OrderNotFoundError struct {
	OrderID      string
	ResponseBody string
}

func (e *OrderNotFoundError) Error() string {
	return fmt.Sprintf("gocuotas: order %q not found (HTTP %d)", e.OrderID, HTTPStatusOrderNotFound)
}

// APIError returns a generic *api.APIError view (same status/body) for uniform handling.
func (e *OrderNotFoundError) APIError() *api.APIError {
	return &api.APIError{
		StatusCode:   HTTPStatusOrderNotFound,
		ResponseBody: e.ResponseBody,
		Message:      e.Error(),
	}
}
