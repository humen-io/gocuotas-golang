package clientv1

import "encoding/json"

// CommerceResponse is the JSON body of GET /api_client/v1/client.
type CommerceResponse struct {
	ID                          int64  `json:"id"`
	Name                        string `json:"name"`
	Cuit                        string `json:"cuit"`
	SurchargePercentageToOnline string `json:"surcharge_percentage_to_online_orders"`
	MaxNumberOfInstallments     int    `json:"max_number_of_installments"`
}

// Liquidacion is one element of GET /api_client/v1/expense_settlements.
type Liquidacion struct {
	ID                                  int64  `json:"id"`
	PaymentExpenseMethod                string `json:"payment_expense_method"`
	PaymentExpenseAt                    string `json:"payment_expense_at"`
	DueExpenseAt                        string `json:"due_expense_at"`
	PaymentExpenseRetainedAmountInCents int64  `json:"payment_expense_retained_amount_in_cents"`
	PaymentExpenseAmountInCents         int64  `json:"payment_expense_amount_in_cents"`
}

// SettlementDetail is one element of the "details" array on GET /api_client/v1/expense_settlements/{id}.
type SettlementDetail struct {
	ID                      int64            `json:"id"`
	Description             string           `json:"description"`
	DeliveredAt             string           `json:"delivered_at"`
	DueExpenseAt            string           `json:"due_expense_at"`
	AmountInCents           int64            `json:"amount_in_cents"`
	CommissionAmountInCents int64            `json:"commission_amount_in_cents"`
	TaxAmountInCents        int64            `json:"tax_amount_in_cents"`
	ExpenseAmountInCents    int64            `json:"expense_amount_in_cents"`
	DiscardedAt             *string          `json:"discarded_at"`
	Status                  string           `json:"status"`
	NumberOfInstallments    int              `json:"number_of_installments"`
	OrderReferenceID        *string          `json:"order_reference_id"`
	Payment                 *LiquidacionPago `json:"payment"`
}

type LiquidacionPago struct {
	Card *LiquidacionTarjeta `json:"card"`
}

type LiquidacionTarjeta struct {
	Number string `json:"number"`
	Name   string `json:"name"`
}

// SettlementInfo is GET /api_client/v1/expense_settlements/{id}.
// Retention arrays are raw JSON to tolerate {} or evolving shapes (same idea as Java's JsonNode list).
type SettlementInfo struct {
	ID                                  int64              `json:"id"`
	PaymentExpenseMethod                string             `json:"payment_expense_method"`
	PaymentExpenseAt                    string             `json:"payment_expense_at"`
	DueExpenseAt                        string             `json:"due_expense_at"`
	PaymentExpenseRetainedAmountInCents int64              `json:"payment_expense_retained_amount_in_cents"`
	PaymentExpenseAmountInCents         int64              `json:"payment_expense_amount_in_cents"`
	Details                             []SettlementDetail `json:"details"`
	NormalRetentionRetainPaidOrders     []json.RawMessage  `json:"normal_retention_retain_paid_orders"`
	TaxRetentionRetainPaidOrders        []json.RawMessage  `json:"tax_retention_retain_paid_orders"`
}
