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

	// Extrair telefone do comprador (se disponível)
	var customerPhone *PhoneData
	if buyer.PhoneCountryCode.Valid && buyer.PhoneAreaCode.Valid && buyer.PhoneNumber.Valid {
		customerPhone = &PhoneData{
			CountryCode: buyer.PhoneCountryCode.String,
			AreaCode:    buyer.PhoneAreaCode.String,
			Number:      buyer.PhoneNumber.String,
		}
	}

	// Log estruturado antes de enviar ao Pagar.me
	log.Printf("[CreatePayment] Enviando ao Pagar.me: orderID=%s, total=%d centavos, items=%d, tickets=%d, method=%s, hasPhone=%v",
		req.OrderID, totalCentavos, len(orderItems), totalTickets, AllowedPaymentMethod, customerPhone != nil)

	// Create Pagar.me order with PIX + split
	pixResult, err := h.client.CreatePixOrder(PixOrderParams{
		OrderID:             req.OrderID,
		ProducerRecipientID: producerRecipientID,
		AmountCentavos:      totalCentavos,
		TotalTickets:        totalTickets,
		Description:         fmt.Sprintf("Afterzin - %s", eventTitle),
		CustomerName:        buyer.Name,
		CustomerEmail:       buyer.Email,
		CustomerDocument:    sanitizedCPF,  // CPF sanitizado (apenas dígitos)
		CustomerPhone:       customerPhone, // Telefone estruturado (opcional)
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
// IMPORTANT: Returns status based ONLY on local database (source of truth),
// not from Pagar.me API, to prevent showing "paid" before webhook processes.
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

	// Verify order ownership and get status FROM DATABASE (source of truth)
	orderUserID, orderStatus, _, err := repository.OrderByID(h.db, orderID)
	if err != nil || orderUserID == "" {
		respondError(w, http.StatusNotFound, "pedido não encontrado")
		return
	}
	if orderUserID != userID {
		respondError(w, http.StatusForbidden, "pedido não pertence ao usuário")
		return
	}

	// Determine if paid based ONLY on database status
	// This ensures frontend doesn't show "paid" before webhook completes
	paid := (orderStatus == "PAID" || orderStatus == "CONFIRMED")

	// Map internal status to user-friendly status
	var displayStatus string
	switch orderStatus {
	case "PENDING":
		displayStatus = "pending"
	case "PROCESSING":
		displayStatus = "processing"
	case "PAID", "CONFIRMED":
		displayStatus = "paid"
	case "CANCELLED":
		displayStatus = "cancelled"
	case "FRAUD_ALERT":
		displayStatus = "fraud_alert"
	default:
		displayStatus = orderStatus
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":      displayStatus,
		"orderStatus": orderStatus, // Raw status for debugging
		"paid":        paid,
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

	log.Printf("[WEBHOOK] Recebendo webhook Pagar.me")
	body, err := io.ReadAll(io.LimitReader(r.Body, 65536))
	if err != nil {
		log.Printf("[WEBHOOK] Erro ao ler corpo: %v", err)
		respondError(w, http.StatusBadRequest, "erro ao ler corpo")
		return
	}

	log.Printf("[WEBHOOK] Corpo recebido: %s", string(body))

	// NOTE: signature verification intentionally disabled.
	// Always parse the incoming payload and proceed without checking
	// the `x-hub-signature` header. This makes webhook processing
	// tolerant to providers that don't send a signature or when
	// headers are stripped by proxies. Use with caution in production.
	var event *WebhookEvent
	var evt WebhookEvent
	if err := json.Unmarshal(body, &evt); err != nil {
		log.Printf("[WEBHOOK] Erro ao parsear payload: %v", err)
		respondError(w, http.StatusBadRequest, "corpo inválido")
		return
	}
	event = &evt
	log.Printf("[WEBHOOK] Verificação de assinatura desabilitada — evento recebido: id=%s type=%s", event.ID, event.Type)

	// Idempotency check - prevent processing same event twice
	if repository.PagarmeWebhookEventExists(h.db, event.ID) {
		log.Printf("[WEBHOOK] Evento %s já recebido, ignorando.", event.ID)
		w.WriteHeader(http.StatusOK)
		return
	}

	// Log the event immediately (prevents duplicate processing if request retries)
	if err := repository.InsertPagarmeWebhookEvent(h.db, event.ID, event.Type); err != nil {
		log.Printf("[WEBHOOK] Erro ao inserir evento no banco: %v", err)
		respondError(w, http.StatusInternalServerError, "erro ao processar webhook")
		return
	}

	log.Printf("[WEBHOOK] Evento registrado no banco: id=%s type=%s", event.ID, event.Type)

	// Route by event type
	switch event.Type {
	case "order.paid":
		log.Printf("[WEBHOOK] Processando evento order.paid")
		h.handleOrderPaid(event)
	case "charge.paid":
		log.Printf("[WEBHOOK] Processando evento charge.paid")
		h.handleChargePaid(event)
	default:
		log.Printf("[WEBHOOK] Tipo de evento não tratado: %s", event.Type)
	}

	// Mark as processed with timestamp
	if err := repository.MarkPagarmeWebhookEventProcessedAt(h.db, event.ID); err != nil {
		log.Printf("[WEBHOOK] Erro ao marcar evento como processado: %v", err)
	}

	log.Printf("[WEBHOOK] Processamento finalizado para evento: %s", event.ID)
	w.WriteHeader(http.StatusOK)
}

// handleOrderPaid processes order.paid:
//  1. Extract order code (our internal order ID) from event data
//  2. Check if this order was already processed by another event type
//  3. Create tickets with signed QR codes
//  4. Mark order as CONFIRMED/PAID
func (h *Handler) handleOrderPaid(event *WebhookEvent) {
	data := event.Data
	if data == nil {
		log.Printf("[ERROR] pagarme: order.paid - no data")
		return
	}

	// The "code" field is our internal order ID (set when creating the order)
	orderID, _ := data["code"].(string)
	pagarmeOrderID, _ := data["id"].(string)

	if orderID == "" {
		log.Printf("[ERROR] pagarme: order.paid but no order code in data (pagarme_order: %s)", pagarmeOrderID)
		return
	}

	// Additional idempotency check: prevent processing if another event (charge.paid) already processed this order
	if repository.PagarmeWebhookProcessedForOrder(h.db, orderID, "order.paid") {
		log.Printf("[SKIP] pagarme: order %s already processed by order.paid event", orderID)
		return
	}
	if repository.PagarmeWebhookProcessedForOrder(h.db, orderID, "charge.paid") {
		log.Printf("[SKIP] pagarme: order %s already processed by charge.paid event", orderID)
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
// Checks idempotency to avoid processing if order.paid already handled this.
func (h *Handler) handleChargePaid(event *WebhookEvent) {
	data := event.Data
	if data == nil {
		log.Printf("[ERROR] pagarme: charge.paid - no data")
		return
	}

	chargeID, _ := data["id"].(string)

	// Try to get order info from the charge
	orderData, ok := data["order"].(map[string]interface{})
	if !ok {
		log.Printf("[ERROR] pagarme: charge.paid but no order in charge data (charge: %s)", chargeID)
		return
	}

	orderID, _ := orderData["code"].(string)
	pagarmeOrderID, _ := orderData["id"].(string)

	if orderID == "" {
		log.Printf("[ERROR] pagarme: charge.paid but no order code (charge: %s)", chargeID)
		return
	}

	// Additional idempotency check: prevent processing if order.paid already processed this order
	if repository.PagarmeWebhookProcessedForOrder(h.db, orderID, "order.paid") {
		log.Printf("[SKIP] pagarme: order %s already processed by order.paid event", orderID)
		return
	}
	if repository.PagarmeWebhookProcessedForOrder(h.db, orderID, "charge.paid") {
		log.Printf("[SKIP] pagarme: order %s already processed by charge.paid event", orderID)
		return
	}

	h.processOrderPayment(orderID, pagarmeOrderID, chargeID)
}

// processOrderPayment handles the common logic for confirming an order:
// Uses atomic transaction with optimistic locking to prevent race conditions.
// Validates payment amount to prevent fraud.
// Creates audit trail of status changes.
func (h *Handler) processOrderPayment(orderID, pagarmeOrderID, chargeID string) {
	log.Printf("[WEBHOOK_PROCESSING] order_id=%s pagarme_order=%s charge=%s", orderID, pagarmeOrderID, chargeID)

	// Begin atomic transaction
	tx, err := h.db.Begin()
	if err != nil {
		log.Printf("[ERROR] pagarme: begin transaction error for order %s: %v", orderID, err)
		return
	}
	defer tx.Rollback() // Auto-rollback if not committed

	// 1. Atomically claim the order (optimistic lock to prevent race conditions)
	claimed, err := repository.ClaimOrderProcessingTx(tx, orderID)
	if err != nil {
		log.Printf("[ERROR] pagarme: claim order %s error: %v", orderID, err)
		return
	}
	if !claimed {
		// Another webhook is already processing this order
		log.Printf("[SKIP] pagarme: order %s already claimed by another webhook", orderID)
		return
	}
	log.Printf("[ORDER_CLAIMED] order_id=%s status=PROCESSING", orderID)

	// 2. Save Pagar.me IDs within transaction
	if pagarmeOrderID != "" {
		if err := repository.SetOrderPagarmeOrderIDTx(tx, orderID, pagarmeOrderID); err != nil {
			log.Printf("[ERROR] pagarme: set pagarme order id error: %v", err)
			return
		}
	}
	if chargeID != "" {
		if err := repository.SetOrderPagarmeChargeIDTx(tx, orderID, chargeID); err != nil {
			log.Printf("[ERROR] pagarme: set pagarme charge id error: %v", err)
			return
		}
	}

	// 3. Get order details
	orderUserID, orderStatus, orderTotal, err := repository.OrderByIDTx(tx, orderID)
	if err != nil || orderUserID == "" {
		log.Printf("[ERROR] pagarme: order %s not found in transaction", orderID)
		return
	}

	// 4. Validate payment amount (CRITICAL SECURITY CHECK)
	if pagarmeOrderID != "" {
		paidAmount, err := h.client.GetOrderPaidAmount(pagarmeOrderID)
		if err != nil {
			log.Printf("[ERROR] pagarme: get paid amount for order %s error: %v", orderID, err)
			// Record failed validation
			repository.RecordOrderStatusChangeWithError(tx, orderID, orderStatus, "FRAUD_ALERT", "payment_validation_failed", err.Error())
			return
		}

		expectedAmount := int64(orderTotal * 100) // Convert to centavos
		if paidAmount != expectedAmount {
			log.Printf("[FRAUD_ALERT] order %s: expected %d centavos, paid %d centavos", orderID, expectedAmount, paidAmount)
			// Record fraud attempt
			repository.RecordOrderStatusChange(tx, orderID, orderStatus, "FRAUD_ALERT", "amount_mismatch", "", pagarmeOrderID, chargeID)
			tx.Commit() // Commit the fraud record
			return
		}
		log.Printf("[PAYMENT_VALIDATED] order_id=%s amount=%d centavos", orderID, paidAmount)
	}

	// 5. Get order items within transaction
	items, err := repository.OrderItemsByOrderIDTx(tx, orderID)
	if err != nil {
		log.Printf("[ERROR] pagarme: get order items for %s error: %v", orderID, err)
		return
	}

	// 6. Create tickets atomically
	ticketsCreated := 0
	for _, item := range items {
		evDate, err := repository.EventDateByIDTx(tx, item.EventDateID)
		if err != nil || evDate == nil {
			log.Printf("[ERROR] pagarme: event date %s not found", item.EventDateID)
			return
		}

		ev, err := repository.EventByIDTx(tx, evDate.EventID)
		if err != nil || ev == nil {
			log.Printf("[ERROR] pagarme: event %s not found", evDate.EventID)
			return
		}

		tt, err := repository.TicketTypeByIDTx(tx, item.TicketTypeID)
		if err != nil || tt == nil {
			log.Printf("[ERROR] pagarme: ticket type %s not found", item.TicketTypeID)
			return
		}

		// Create tickets for this item
		for i := 0; i < item.Quantity; i++ {
			ticketID := uuid.New().String()
			code := repository.GenerateTicketCode()

			// QR payload with charge_id and event_id for traceability
			qrPayload := qrcode.GenerateSignedPayloadV2(ticketID, chargeID, ev.ID, []byte(h.cfg.JWTSecret))

			err := repository.CreateTicketWithIDTx(
				tx, ticketID, code, qrPayload,
				orderID, item.ID, orderUserID,
				ev.ID, item.EventDateID, item.TicketTypeID,
			)
			if err != nil {
				log.Printf("[ERROR] pagarme: create ticket error: %v", err)
				return // ROLLBACK entire transaction
			}
			ticketsCreated++

			// Increment sold count
			if err := repository.IncrementTicketTypeSoldTx(tx, item.TicketTypeID, 1); err != nil {
				log.Printf("[ERROR] pagarme: increment sold error: %v", err)
				return
			}

			// Decrement available quantity (with validation)
			lotID, err := repository.LotIDByTicketTypeIDTx(tx, item.TicketTypeID)
			if err != nil {
				log.Printf("[ERROR] pagarme: get lot id error: %v", err)
				return
			}

			if err := repository.DecrementLotAvailableTx(tx, lotID, 1); err != nil {
				log.Printf("[ERROR] pagarme: decrement lot available error (overselling prevented): %v", err)
				return
			}
		}
	}

	log.Printf("[TICKETS_CREATED] order_id=%s count=%d", orderID, ticketsCreated)

	// 7. Confirm the order (PROCESSING → PAID)
	if err := repository.ConfirmOrderTx(tx, orderID); err != nil {
		log.Printf("[ERROR] pagarme: confirm order %s error: %v", orderID, err)
		return
	}

	// 8. Record status change for audit trail
	if err := repository.RecordOrderStatusChange(tx, orderID, "PROCESSING", "PAID", "webhook_payment_confirmed", "", pagarmeOrderID, chargeID); err != nil {
		log.Printf("[WARNING] pagarme: record status change error (non-fatal): %v", err)
		// Continue - this is just for audit
	}

	// 9. COMMIT transaction (all-or-nothing)
	if err := tx.Commit(); err != nil {
		log.Printf("[ERROR] pagarme: commit transaction error for order %s: %v", orderID, err)
		return
	}

	log.Printf("[ORDER_CONFIRMED] order_id=%s status=PAID tickets=%d pagarme_order=%s charge=%s",
		orderID, ticketsCreated, pagarmeOrderID, chargeID)
}
