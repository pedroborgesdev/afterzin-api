package pagarme

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"

	"afterzin/api/internal/config"
	"afterzin/api/internal/middleware"
	"afterzin/api/internal/qrcode"
	"afterzin/api/internal/repository"

	"github.com/google/uuid"
)

// Handler provides HTTP handlers for Pagar.me REST endpoints.
// These complement the GraphQL API with payment-specific operations
// that are naturally REST (webhooks, PIX flow, etc.).
type Handler struct {
	client *Client
	db     *sql.DB
	cfg    *config.Config
}

// NewHandler creates a new Pagar.me HTTP handler.
func NewHandler(client *Client, db *sql.DB, cfg *config.Config) *Handler {
	return &Handler{client: client, db: db, cfg: cfg}
}

func respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func respondError(w http.ResponseWriter, status int, message string) {
	respondJSON(w, status, map[string]string{"error": message})
}

// sanitizeDocument remove todos os caracteres não numéricos de um documento (CPF/CNPJ).
func sanitizeDocument(doc string) string {
	return regexp.MustCompile(`[^\d]`).ReplaceAllString(doc, "")
}

// ---------- Recipient Management ----------

// CreateRecipient handles POST /api/pagarme/recipient/create
// Creates a Pagar.me recipient for the authenticated producer using bank account data.
func (h *Handler) CreateRecipient(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := middleware.UserID(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	// Get or create producer profile
	prodID, _ := repository.ProducerIDByUser(h.db, userID)
	if prodID == "" {
		var err error
		prodID, err = repository.CreateProducer(h.db, userID)
		if err != nil {
			respondError(w, http.StatusInternalServerError, "erro ao criar perfil de produtor")
			return
		}
	}

	// Check if already has a recipient
	existing, _ := repository.GetProducerPagarmeRecipientID(h.db, prodID)
	if existing != "" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"recipientId": existing,
			"message":     "recebedor Pagar.me já existe",
		})
		return
	}

	// Parse request body
	var req struct {
		Document     string `json:"document"`
		DocumentType string `json:"documentType"`
		Type         string `json:"type"`
		// PF
		Name                   string   `json:"name"`
		Email                  string   `json:"email"`
		Phone                  string   `json:"phone"`
		Birthdate              string   `json:"birthdate"`
		MonthlyIncome          int      `json:"monthly_income"`
		ProfessionalOccupation string   `json:"professional_occupation"`
		Address                *Address `json:"address"`
		// PJ
		CompanyName   string `json:"company_name"`
		TradingName   string `json:"trading_name"`
		AnnualRevenue int    `json:"annual_revenue"`
		// PJ também pode ter telefone
		// Bank
		BankCode          string `json:"bankCode"`
		BranchNumber      string `json:"branchNumber"`
		BranchCheckDigit  string `json:"branchCheckDigit"`
		AccountNumber     string `json:"accountNumber"`
		AccountCheckDigit string `json:"accountCheckDigit"`
		AccountType       string `json:"accountType"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "corpo inválido")
		return
	}

	if req.Document == "" || req.BankCode == "" || req.BranchNumber == "" || req.AccountNumber == "" {
		respondError(w, http.StatusBadRequest, "documento, banco, agência e conta são obrigatórios")
		return
	}

	// Default values
	if req.DocumentType == "" {
		req.DocumentType = "CPF"
	}
	if req.Type == "" {
		req.Type = "individual"
	}
	if req.AccountType == "" {
		req.AccountType = "checking"
	}

	// Get user info
	user, _ := repository.UserByID(h.db, userID)
	if user == nil {
		respondError(w, http.StatusInternalServerError, "usuário não encontrado")
		return
	}

	// Create recipient in Pagar.me
	result, err := h.client.CreateRecipient(CreateRecipientParams{
		Name:                   req.Name,
		Email:                  req.Email,
		Phone:                  req.Phone,
		Document:               req.Document,
		DocumentType:           req.DocumentType,
		Type:                   req.Type,
		Birthdate:              req.Birthdate,
		MonthlyIncome:          req.MonthlyIncome,
		ProfessionalOccupation: req.ProfessionalOccupation,
		Address:                req.Address,
		CompanyName:            req.CompanyName,
		TradingName:            req.TradingName,
		AnnualRevenue:          req.AnnualRevenue,
		BankCode:               req.BankCode,
		BranchNumber:           req.BranchNumber,
		BranchCheckDigit:       req.BranchCheckDigit,
		AccountNumber:          req.AccountNumber,
		AccountCheckDigit:      req.AccountCheckDigit,
		AccountType:            req.AccountType,
	})
	if err != nil {
		log.Printf("pagarme: create recipient error: %v", err)
		respondError(w, http.StatusInternalServerError, "erro ao criar recebedor: "+err.Error())
		return
	}

	// Persist recipient ID
	if err := repository.SetProducerPagarmeRecipientID(h.db, prodID, result.RecipientID); err != nil {
		log.Printf("pagarme: save recipient id error: %v", err)
		respondError(w, http.StatusInternalServerError, "erro ao salvar recebedor")
		return
	}

	// Mark onboarding as complete
	repository.SetProducerOnboardingComplete(h.db, prodID, true)

	log.Printf("pagarme: recipient created for producer %s (recipient: %s)", prodID, result.RecipientID)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"recipientId": result.RecipientID,
		"status":      result.Status,
		"message":     "recebedor criado com sucesso",
	})
}

// GetRecipientStatus handles GET /api/pagarme/recipient/status
// Returns the current recipient status of the producer.
func (h *Handler) GetRecipientStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := middleware.UserID(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	prodID, _ := repository.ProducerIDByUser(h.db, userID)
	if prodID == "" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"hasRecipient":       false,
			"onboardingComplete": false,
		})
		return
	}

	recipientID, _ := repository.GetProducerPagarmeRecipientID(h.db, prodID)
	if recipientID == "" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"hasRecipient":       false,
			"onboardingComplete": false,
		})
		return
	}

	// Check live status from Pagar.me
	recipientData, err := h.client.GetRecipient(recipientID)
	if err != nil {
		log.Printf("pagarme: get recipient status error: %v", err)
		// Return cached local status
		onboardingComplete, _ := repository.GetProducerOnboardingComplete(h.db, prodID)
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"hasRecipient":       true,
			"recipientId":        recipientID,
			"onboardingComplete": onboardingComplete,
			"error":              "não foi possível verificar status com Pagar.me",
		})
		return
	}

	status, _ := recipientData["status"].(string)
	name, _ := recipientData["name"].(string)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"hasRecipient":       true,
		"recipientId":        recipientID,
		"onboardingComplete": true,
		"status":             status,
		"name":               name,
	})
}

// ---------- Payment: PIX via Pagar.me ----------

// CreatePayment handles POST /api/pagarme/payment/create
// Creates a Pagar.me order with PIX payment and split for an existing order.
// Returns QR code + copia-e-cola for the customer to pay.
func (h *Handler) CreatePayment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := middleware.UserID(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	var req struct {
		OrderID string `json:"orderId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	if req.OrderID == "" {
		respondError(w, http.StatusBadRequest, "orderId é obrigatório")
		return
	}

	// Verify order ownership and status
	orderUserID, status, _, err := repository.OrderByID(h.db, req.OrderID)
	if err != nil || orderUserID == "" {
		respondError(w, http.StatusNotFound, "pedido não encontrado")
		return
	}
	if orderUserID != userID {
		respondError(w, http.StatusForbidden, "pedido não pertence ao usuário")
		return
	}
	if status != "PENDING" {
		respondError(w, http.StatusBadRequest, "pedido já processado")
		return
	}

	// Check if order already has a Pagar.me order (avoid duplicate charges)
	existingOrderID, _ := repository.GetOrderPagarmeOrderID(h.db, req.OrderID)
	if existingOrderID != "" {
		// Return existing order status
		orderStatus, err := h.client.GetOrderStatus(existingOrderID)
		if err == nil && orderStatus.Status != "canceled" && orderStatus.Status != "failed" {
			respondJSON(w, http.StatusOK, orderStatus)
			return
		}
		// If cancelled or errored, allow creating a new one
	}

	// Get order items
	items, err := repository.OrderItemsByOrderID(h.db, req.OrderID)
	if err != nil || len(items) == 0 {
		respondError(w, http.StatusBadRequest, "pedido sem itens")
		return
	}

	// Get customer (buyer) info
	buyer, _ := repository.UserByID(h.db, userID)
	if buyer == nil {
		respondError(w, http.StatusInternalServerError, "usuário não encontrado")
		return
	}

	// Sanitizar e validar CPF do comprador
	sanitizedCPF := sanitizeDocument(buyer.CPF)
	if len(sanitizedCPF) != 11 {
		respondError(w, http.StatusBadRequest, fmt.Sprintf("CPF inválido: deve conter 11 dígitos (recebido %d)", len(sanitizedCPF)))
		return
	}

	// Calculate total amount, resolve producer recipient, build order items
	var producerRecipientID string
	var totalCentavos int64
	var totalTickets int
	var eventTitle string
	var orderItems []OrderItem

	for _, item := range items {
		// Validar quantidade
		if item.Quantity <= 0 {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("quantidade do item deve ser maior que zero (item: %s)", item.TicketTypeID))
			return
		}

		totalTickets += item.Quantity

		tt, _ := repository.TicketTypeByID(h.db, item.TicketTypeID)
		if tt == nil {
			respondError(w, http.StatusBadRequest, "tipo de ingresso não encontrado")
			return
		}

		// Validar preço unitário
		if tt.Price <= 0 {
			respondError(w, http.StatusBadRequest, fmt.Sprintf("preço unitário deve ser maior que zero (ticket: %s)", item.TicketTypeID))
			return
		}

		totalCentavos += int64(tt.Price*100) * int64(item.Quantity)

		// Resolve event → producer → recipient
		ed, _ := repository.EventDateByID(h.db, item.EventDateID)
		if ed == nil {
			respondError(w, http.StatusBadRequest, "data do evento não encontrada")
			return
		}
		ev, _ := repository.EventByID(h.db, ed.EventID)
		if ev == nil {
			respondError(w, http.StatusBadRequest, "evento não encontrado")
			return
		}
		if eventTitle == "" {
			eventTitle = ev.Title
		}

		if producerRecipientID == "" {
			recipientID, _ := repository.GetProducerPagarmeRecipientID(h.db, ev.ProducerID)
			if recipientID == "" {
				respondError(w, http.StatusBadRequest, "produtor não configurou recebimento de pagamentos")
				return
			}
			producerRecipientID = recipientID
		}

		orderItems = append(orderItems, OrderItem{
			Code:        item.TicketTypeID,
			Description: fmt.Sprintf("%s - %s", tt.Name, eventTitle),
			Quantity:    item.Quantity,
			Amount:      int64(tt.Price * 100), // unit price in centavos
		})
	}

	// Validar valor total
	if totalCentavos <= 0 {
		respondError(w, http.StatusBadRequest, "valor total deve ser maior que zero")
		return
	}

	// Log estruturado antes de enviar ao Pagar.me
	log.Printf("[CreatePayment] Enviando ao Pagar.me: orderID=%s, total=%d centavos, items=%d, tickets=%d, method=%s",
		req.OrderID, totalCentavos, len(orderItems), totalTickets, AllowedPaymentMethod)

	// Create Pagar.me order with PIX + split
	pixResult, err := h.client.CreatePixOrder(PixOrderParams{
		OrderID:             req.OrderID,
		ProducerRecipientID: producerRecipientID,
		AmountCentavos:      totalCentavos,
		TotalTickets:        totalTickets,
		Description:         fmt.Sprintf("Afterzin - %s", eventTitle),
		CustomerName:        buyer.Name,
		CustomerEmail:       buyer.Email,
		CustomerDocument:    sanitizedCPF, // CPF sanitizado (apenas dígitos)
		Items:               orderItems,
	})
	if err != nil {
		log.Printf("pagarme: create pix order error: %v", err)
		respondError(w, http.StatusInternalServerError, "erro ao criar pagamento PIX: "+err.Error())
		return
	}

	// Persist Pagar.me IDs on order
	repository.SetOrderPagarmeOrderID(h.db, req.OrderID, pixResult.PagarmeOrderID)
	repository.SetOrderPagarmeChargeID(h.db, req.OrderID, pixResult.PagarmeChargeID)

	log.Printf("pagarme: PIX order created for order %s (pagarme_order: %s, charge: %s, amount: %d, fee: %d×%d)",
		req.OrderID, pixResult.PagarmeOrderID, pixResult.PagarmeChargeID,
		totalCentavos, h.client.ApplicationFee, totalTickets)

	respondJSON(w, http.StatusOK, pixResult)
}

// GetPaymentStatus handles GET /api/pagarme/payment/status?orderId=xxx
// Frontend polls this to check if PIX was paid.
func (h *Handler) GetPaymentStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	userID := middleware.UserID(r.Context())
	if userID == "" {
		respondError(w, http.StatusUnauthorized, "não autenticado")
		return
	}

	orderID := r.URL.Query().Get("orderId")
	if orderID == "" {
		respondError(w, http.StatusBadRequest, "orderId é obrigatório")
		return
	}

	// Verify order ownership
	orderUserID, orderStatus, _, err := repository.OrderByID(h.db, orderID)
	if err != nil || orderUserID == "" {
		respondError(w, http.StatusNotFound, "pedido não encontrado")
		return
	}
	if orderUserID != userID {
		respondError(w, http.StatusForbidden, "pedido não pertence ao usuário")
		return
	}

	// If order is already confirmed, return immediately
	if orderStatus == "CONFIRMED" || orderStatus == "PAID" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status":      "paid",
			"orderStatus": orderStatus,
			"paid":        true,
		})
		return
	}

	// Get Pagar.me order ID and check status
	pagarmeOrderID, _ := repository.GetOrderPagarmeOrderID(h.db, orderID)
	if pagarmeOrderID == "" {
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "no_payment",
			"paid":   false,
		})
		return
	}

	pagarmeStatus, err := h.client.GetOrderStatus(pagarmeOrderID)
	if err != nil {
		log.Printf("pagarme: get order status error: %v", err)
		respondJSON(w, http.StatusOK, map[string]interface{}{
			"status": "unknown",
			"paid":   false,
			"error":  "não foi possível verificar status",
		})
		return
	}

	paid := pagarmeStatus.Status == "paid"

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":         pagarmeStatus.Status,
		"pagarmeOrderId": pagarmeStatus.PagarmeOrderID,
		"paid":           paid,
	})
}

// ---------- Webhooks ----------

// HandleWebhook handles POST /api/pagarme/webhook
// Verifies signature, deduplicates, and processes Pagar.me events.
//
// Handled events:
//   - order.paid → confirms order, creates tickets, generates QR codes
//   - charge.paid → fallback handler
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		respondError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		respondError(w, http.StatusBadRequest, "erro ao ler corpo")
		return
	}

	sigHeader := r.Header.Get("x-hub-signature")
	if sigHeader == "" {
		respondError(w, http.StatusBadRequest, "assinatura ausente")
		return
	}

	event, err := h.client.VerifyWebhookSignature(body, sigHeader)
	if err != nil {
		log.Printf("pagarme: webhook signature error: %v", err)
		respondError(w, http.StatusBadRequest, "assinatura inválida")
		return
	}

	// Idempotency check
	if repository.PagarmeWebhookEventExists(h.db, event.ID) {
		w.WriteHeader(http.StatusOK)
		return
	}

	// Log the event
	repository.InsertPagarmeWebhookEvent(h.db, event.ID, event.Type)

	// Route by event type
	switch event.Type {
	case "order.paid":
		h.handleOrderPaid(event)
	case "charge.paid":
		h.handleChargePaid(event)
	default:
		log.Printf("pagarme: unhandled webhook event type: %s", event.Type)
	}

	// Mark as processed
	repository.MarkPagarmeWebhookEventProcessed(h.db, event.ID)

	w.WriteHeader(http.StatusOK)
}

// handleOrderPaid processes order.paid:
//  1. Extract order code (our internal order ID) from event data
//  2. Create tickets with signed QR codes
//  3. Mark order as CONFIRMED/PAID
func (h *Handler) handleOrderPaid(event *WebhookEvent) {
	data := event.Data
	if data == nil {
		log.Printf("pagarme: order.paid - no data")
		return
	}

	// The "code" field is our internal order ID (set when creating the order)
	orderID, _ := data["code"].(string)
	pagarmeOrderID, _ := data["id"].(string)

	if orderID == "" {
		log.Printf("pagarme: order.paid but no order code in data (pagarme_order: %s)", pagarmeOrderID)
		return
	}

	// Extract charge ID for QR code traceability
	chargeID := ""
	if charges, ok := data["charges"].([]interface{}); ok && len(charges) > 0 {
		if charge, ok := charges[0].(map[string]interface{}); ok {
			chargeID, _ = charge["id"].(string)
		}
	}

	h.processOrderPayment(orderID, pagarmeOrderID, chargeID)
}

// handleChargePaid processes charge.paid as a fallback.
// Tries to extract the order code from the charge's order reference.
func (h *Handler) handleChargePaid(event *WebhookEvent) {
	data := event.Data
	if data == nil {
		log.Printf("pagarme: charge.paid - no data")
		return
	}

	chargeID, _ := data["id"].(string)

	// Try to get order info from the charge
	orderData, ok := data["order"].(map[string]interface{})
	if !ok {
		log.Printf("pagarme: charge.paid but no order in charge data (charge: %s)", chargeID)
		return
	}

	orderID, _ := orderData["code"].(string)
	pagarmeOrderID, _ := orderData["id"].(string)

	if orderID == "" {
		log.Printf("pagarme: charge.paid but no order code (charge: %s)", chargeID)
		return
	}

	h.processOrderPayment(orderID, pagarmeOrderID, chargeID)
}

// processOrderPayment handles the common logic for confirming an order:
// verify pending, create tickets, confirm order.
func (h *Handler) processOrderPayment(orderID, pagarmeOrderID, chargeID string) {
	// Save Pagar.me IDs to order
	if pagarmeOrderID != "" {
		repository.SetOrderPagarmeOrderID(h.db, orderID, pagarmeOrderID)
	}
	if chargeID != "" {
		repository.SetOrderPagarmeChargeID(h.db, orderID, chargeID)
	}

	// Verify order is still pending (idempotency)
	orderUserID, status, _, err := repository.OrderByID(h.db, orderID)
	if err != nil || orderUserID == "" {
		log.Printf("pagarme: order %s not found", orderID)
		return
	}
	if status != "PENDING" {
		log.Printf("pagarme: order %s already %s, skipping ticket creation", orderID, status)
		return
	}

	// Create tickets for each order item
	items, err := repository.OrderItemsByOrderID(h.db, orderID)
	if err != nil {
		log.Printf("pagarme: get order items for %s error: %v", orderID, err)
		return
	}

	for _, item := range items {
		evDate, _ := repository.EventDateByID(h.db, item.EventDateID)
		if evDate == nil {
			continue
		}
		ev, _ := repository.EventByID(h.db, evDate.EventID)
		if ev == nil {
			continue
		}

		for i := 0; i < item.Quantity; i++ {
			ticketID := uuid.New().String()
			code := repository.GenerateTicketCode()

			// QR payload with charge_id and event_id for traceability
			qrPayload := qrcode.GenerateSignedPayloadV2(ticketID, chargeID, ev.ID, []byte(h.cfg.JWTSecret))

			err := repository.CreateTicketWithID(
				h.db, ticketID, code, qrPayload,
				orderID, item.ID, orderUserID,
				ev.ID, item.EventDateID, item.TicketTypeID,
			)
			if err != nil {
				log.Printf("pagarme: create ticket error: %v", err)
				continue
			}
			repository.IncrementTicketTypeSold(h.db, item.TicketTypeID, 1)
			lotID, _ := repository.LotIDByTicketTypeID(h.db, item.TicketTypeID)
			repository.DecrementLotAvailable(h.db, lotID, 1)
		}
	}

	// Confirm the order
	if err := repository.ConfirmOrder(h.db, orderID); err != nil {
		log.Printf("pagarme: confirm order %s error: %v", orderID, err)
	}

	log.Printf("pagarme: order %s CONFIRMED via webhook (pagarme_order: %s, charge: %s)", orderID, pagarmeOrderID, chargeID)
}
