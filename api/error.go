package api

// APIError is the Go equivalent of Java's GoCuotasApiException: non-2xx HTTP from GoCuotas APIs.
type APIError struct {
	StatusCode   int
	ResponseBody string
	Message      string
}

func (e *APIError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	return "gocuotas: HTTP error"
}
