package redirect

import "encoding/json"

// AuthenticationRequest is the JSON body for POST /api_redirect/v1/authentication.
type AuthenticationRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// AuthenticationResponse holds a JWT from authenticate; UnmarshalJSON accepts token, access_token, accessToken, jwt.
type AuthenticationResponse struct {
	Token string
}

func (a *AuthenticationResponse) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	for _, key := range []string{"token", "access_token", "accessToken", "jwt"} {
		v, ok := raw[key]
		if !ok {
			continue
		}
		var s string
		if err := json.Unmarshal(v, &s); err != nil {
			continue
		}
		if s != "" {
			a.Token = s
			return nil
		}
	}
	return nil
}

// CreateCheckoutRequest is the JSON body for POST /api_redirect/v1/checkouts.
type CreateCheckoutRequest struct {
	AmountInCents    int64   `json:"amount_in_cents"`
	Email            string  `json:"email"`
	OrderReferenceID string  `json:"order_reference_id"`
	PhoneNumber      string  `json:"phone_number"`
	URLSuccess       string  `json:"url_success"`
	URLFailure       string  `json:"url_failure"`
	WebhookURL       *string `json:"webhook_url,omitempty"`
}

// CheckoutBuilder mirrors Java's CreateCheckoutRequest.builder().
type CheckoutBuilder struct {
	r CreateCheckoutRequest
}

func NewCheckoutBuilder() *CheckoutBuilder {
	return &CheckoutBuilder{}
}

func (b *CheckoutBuilder) AmountInCents(v int64) *CheckoutBuilder {
	b.r.AmountInCents = v
	return b
}
func (b *CheckoutBuilder) Email(v string) *CheckoutBuilder {
	b.r.Email = v
	return b
}
func (b *CheckoutBuilder) OrderReferenceID(v string) *CheckoutBuilder {
	b.r.OrderReferenceID = v
	return b
}
func (b *CheckoutBuilder) PhoneNumber(v string) *CheckoutBuilder {
	b.r.PhoneNumber = v
	return b
}
func (b *CheckoutBuilder) URLSuccess(v string) *CheckoutBuilder {
	b.r.URLSuccess = v
	return b
}
func (b *CheckoutBuilder) URLFailure(v string) *CheckoutBuilder {
	b.r.URLFailure = v
	return b
}
func (b *CheckoutBuilder) WebhookURL(v *string) *CheckoutBuilder {
	b.r.WebhookURL = v
	return b
}

func (b *CheckoutBuilder) Build() CreateCheckoutRequest {
	return b.r
}

// CreateCheckoutResponse is returned by POST checkouts.
type CreateCheckoutResponse struct {
	URLInit string `json:"url_init"`
}
