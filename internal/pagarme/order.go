package pagarme

import "fmt"

// PixOrderParams holds parameters for creating a Pagar.me order with PIX.
type PixOrderParams struct {
	OrderID             string      // Internal order ID (used as order "code" in Pagar.me)
	ProducerRecipientID string      // Producer's Pagar.me recipient ID (for split)
	AmountCentavos      int64       // Total amount in BRL centavos
	TotalTickets        int         // Ticket count for fee calculation
	Description         string      // Description for the payment
	CustomerName        string      // Buyer's name
	CustomerEmail       string      // Buyer's email
	CustomerDocument    string      // Buyer's CPF
	CustomerPhone       *PhoneData  // Buyer's phone (optional for backward compatibility)
	Items               []OrderItem // Line items
}

// OrderItem represents a single line item in the order.
type OrderItem struct {
	Code        string // ticket_type_id
	Description string // "Ticket Name - Event Name"
	Quantity    int
	Amount      int64 // unit price in centavos
}

// PixOrderResult contains the Pagar.me order data needed by the frontend.
type PixOrderResult struct {
	PagarmeOrderID  string `json:"pagarmeOrderId"`
	PagarmeChargeID string `json:"pagarmeChargeId"`
	PixQRCode       string `json:"pixQrCode"`    // PIX copia-e-cola string
	PixQRCodeURL    string `json:"pixQrCodeUrl"` // URL to QR code image
	ExpiresAt       string `json:"expiresAt"`    // ISO timestamp when PIX expires
	Status          string `json:"status"`       // pending, paid, etc.
}

// CreatePixOrder creates a Pagar.me order with PIX payment method and split.
//
// Split logic:
//   - Platform (Afterzin) receives ApplicationFee × TotalTickets (R$5.00 per ticket default)
//   - Producer receives the remainder
//   - Processing fees are charged to the producer
//
// PIX flow:
//  1. Create order with PIX payment + split
//  2. Pagar.me generates QR code + copia-e-cola
//  3. Return QR data to frontend for display
//  4. Customer scans/pastes in banking app
//  5. Webhook order.paid fires when payment is confirmed
func (c *Client) CreatePixOrder(params PixOrderParams) (*PixOrderResult, error) {
	// Validar parâmetros obrigatórios
	if params.AmountCentavos <= 0 {
		return nil, fmt.Errorf("amount deve ser maior que zero (recebido: %d)", params.AmountCentavos)
	}
	if params.CustomerDocument == "" {
		return nil, fmt.Errorf("documento do cliente é obrigatório")
	}
	if len(params.Items) == 0 {
		return nil, fmt.Errorf("items não pode estar vazio")
	}
	if params.CustomerName == "" {
		return nil, fmt.Errorf("nome do cliente é obrigatório")
	}
	if params.CustomerEmail == "" {
		return nil, fmt.Errorf("email do cliente é obrigatório")
	}

	// Calculate split amounts
	platformFee := c.ApplicationFee * int64(params.TotalTickets)
	producerAmount := params.AmountCentavos - platformFee
	if producerAmount < 0 {
		producerAmount = 0
	}

	// Build items array
	items := make([]map[string]interface{}, len(params.Items))
	for i, item := range params.Items {
		items[i] = map[string]interface{}{
			"code":        item.Code,
			"description": item.Description,
			"quantity":    item.Quantity,
			"amount":      item.Amount,
		}
	}

	// Build split array
	split := []map[string]interface{}{
		{
			"recipient_id": params.ProducerRecipientID,
			"amount":       producerAmount,
			"type":         "flat",
			"liable":       true,
			"options": map[string]interface{}{
				"charge_processing_fee": true,
				"charge_remainder_fee":  true,
			},
		},
	}

	// Only add platform split if PlatformRecipientID is set and fee > 0
	if c.PlatformRecipientID != "" && platformFee > 0 {
		split = append(split, map[string]interface{}{
			"recipient_id": c.PlatformRecipientID,
			"amount":       platformFee,
			"type":         "flat",
			"liable":       false,
			"options": map[string]interface{}{
				"charge_processing_fee": false,
				"charge_remainder_fee":  false,
			},
		})
	}

	// Build customer data
	customer := map[string]interface{}{
		"name":          params.CustomerName,
		"email":         params.CustomerEmail,
		"document":      params.CustomerDocument,
		"document_type": AllowedDocumentType, // Apenas CPF é aceito
		"type":          AllowedCustomerType, // Apenas pessoa física
	}

	// Add phone if available
	if params.CustomerPhone != nil {
		customer["phones"] = map[string]interface{}{
			"mobile_phone": map[string]interface{}{
				"country_code": params.CustomerPhone.CountryCode,
				"area_code":    params.CustomerPhone.AreaCode,
				"number":       params.CustomerPhone.Number,
			},
		}
	}

	body := map[string]interface{}{
		"code":     params.OrderID,
		"customer": customer,
		"items":    items,
		"payments": []map[string]interface{}{
			{
				"payment_method": AllowedPaymentMethod, // Apenas PIX é permitido
				"pix": map[string]interface{}{
					"expires_in": PixExpirationSeconds, // 15 minutos (900 segundos)
				},
				"amount": params.AmountCentavos,
				"split":  split,
			},
		},
	}

	result, err := c.doRequest("POST", "/orders", body)
	if err != nil {
		return nil, fmt.Errorf("create pix order: %w", err)
	}

	orderID, _ := result["id"].(string)
	orderStatus, _ := result["status"].(string)
	if orderID == "" {
		return nil, fmt.Errorf("no order id in response")
	}

	pixResult := &PixOrderResult{
		PagarmeOrderID: orderID,
		Status:         orderStatus,
	}

	// Extract charge and PIX data from response
	extractChargeData(result, pixResult)

	return pixResult, nil
}

// GetOrder retrieves a Pagar.me order by its ID.
func (c *Client) GetOrder(pagarmeOrderID string) (map[string]interface{}, error) {
	return c.doRequest("GET", "/orders/"+pagarmeOrderID, nil)
}

// GetOrderStatus retrieves a Pagar.me order and returns a simplified status.
// Used for frontend polling while waiting for PIX payment.
func (c *Client) GetOrderStatus(pagarmeOrderID string) (*PixOrderResult, error) {
	result, err := c.GetOrder(pagarmeOrderID)
	if err != nil {
		return nil, fmt.Errorf("get order: %w", err)
	}

	orderID, _ := result["id"].(string)
	status, _ := result["status"].(string)

	pixResult := &PixOrderResult{
		PagarmeOrderID: orderID,
		Status:         status,
	}

	// Re-extract charge data (PIX info may still be present if pending)
	extractChargeData(result, pixResult)

	return pixResult, nil
}

// extractChargeData extracts charge ID and PIX transaction data from a Pagar.me order response.
func extractChargeData(result map[string]interface{}, pixResult *PixOrderResult) {
	charges, ok := result["charges"].([]interface{})
	if !ok || len(charges) == 0 {
		return
	}
	charge, ok := charges[0].(map[string]interface{})
	if !ok {
		return
	}
	pixResult.PagarmeChargeID, _ = charge["id"].(string)

	lastTxn, ok := charge["last_transaction"].(map[string]interface{})
	if !ok {
		return
	}
	pixResult.PixQRCode, _ = lastTxn["qr_code"].(string)
	pixResult.PixQRCodeURL, _ = lastTxn["qr_code_url"].(string)
	pixResult.ExpiresAt, _ = lastTxn["expires_at"].(string)
}
