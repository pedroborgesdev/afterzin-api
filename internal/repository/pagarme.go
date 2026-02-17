package repository

import (
	"database/sql"

	"github.com/google/uuid"
)

// ---------- Producer Pagar.me fields ----------

// GetProducerPagarmeRecipientID returns the Pagar.me recipient ID for a producer.
func GetProducerPagarmeRecipientID(db *sql.DB, producerID string) (string, error) {
	var recipientID sql.NullString
	err := db.QueryRow(`SELECT pagarme_recipient_id FROM producers WHERE id = ?`, producerID).Scan(&recipientID)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return recipientID.String, nil
}

// SetProducerPagarmeRecipientID saves the Pagar.me recipient ID for a producer.
func SetProducerPagarmeRecipientID(db *sql.DB, producerID, recipientID string) error {
	_, err := db.Exec(`UPDATE producers SET pagarme_recipient_id = ? WHERE id = ?`, recipientID, producerID)
	return err
}

// GetProducerOnboardingComplete returns whether the producer has completed payment onboarding.
// Reuses the stripe_onboarding_complete column (shared concept).
func GetProducerOnboardingComplete(db *sql.DB, producerID string) (bool, error) {
	var complete int
	err := db.QueryRow(`SELECT stripe_onboarding_complete FROM producers WHERE id = ?`, producerID).Scan(&complete)
	if err != nil {
		return false, err
	}
	return complete == 1, nil
}

// SetProducerOnboardingComplete updates the onboarding complete flag.
func SetProducerOnboardingComplete(db *sql.DB, producerID string, complete bool) error {
	v := 0
	if complete {
		v = 1
	}
	_, err := db.Exec(`UPDATE producers SET stripe_onboarding_complete = ? WHERE id = ?`, v, producerID)
	return err
}

// ---------- Order Pagar.me fields ----------

// SetOrderPagarmeOrderID saves the Pagar.me order ID on an order.
func SetOrderPagarmeOrderID(db *sql.DB, orderID, pagarmeOrderID string) error {
	_, err := db.Exec(`UPDATE orders SET pagarme_order_id = ? WHERE id = ?`, pagarmeOrderID, orderID)
	return err
}

// GetOrderPagarmeOrderID retrieves the Pagar.me order ID for an order.
func GetOrderPagarmeOrderID(db *sql.DB, orderID string) (string, error) {
	var pgOrderID sql.NullString
	err := db.QueryRow(`SELECT pagarme_order_id FROM orders WHERE id = ?`, orderID).Scan(&pgOrderID)
	if err != nil {
		return "", err
	}
	return pgOrderID.String, nil
}

// SetOrderPagarmeChargeID saves the Pagar.me charge ID on an order.
func SetOrderPagarmeChargeID(db *sql.DB, orderID, chargeID string) error {
	_, err := db.Exec(`UPDATE orders SET pagarme_charge_id = ? WHERE id = ?`, chargeID, orderID)
	return err
}

// GetOrderPagarmeChargeID retrieves the Pagar.me charge ID for an order.
func GetOrderPagarmeChargeID(db *sql.DB, orderID string) (string, error) {
	var chargeID sql.NullString
	err := db.QueryRow(`SELECT pagarme_charge_id FROM orders WHERE id = ?`, orderID).Scan(&chargeID)
	if err != nil {
		return "", err
	}
	return chargeID.String, nil
}

// ---------- Pagar.me Webhook Events ----------

// PagarmeWebhookEventExists checks if a Pagar.me webhook event has already been received.
func PagarmeWebhookEventExists(db *sql.DB, eventID string) bool {
	var exists int
	err := db.QueryRow(`SELECT COUNT(*) FROM pagarme_webhook_events WHERE pagarme_event_id = ?`, eventID).Scan(&exists)
	return err == nil && exists > 0
}

// InsertPagarmeWebhookEvent logs a received Pagar.me webhook event.
func InsertPagarmeWebhookEvent(db *sql.DB, eventID, eventType string) error {
	id := uuid.New().String()
	_, err := db.Exec(
		`INSERT OR IGNORE INTO pagarme_webhook_events (id, pagarme_event_id, event_type) VALUES (?, ?, ?)`,
		id, eventID, eventType,
	)
	return err
}

// MarkPagarmeWebhookEventProcessed marks a Pagar.me webhook event as successfully processed.
func MarkPagarmeWebhookEventProcessed(db *sql.DB, eventID string) error {
	_, err := db.Exec(`UPDATE pagarme_webhook_events SET processed = 1 WHERE pagarme_event_id = ?`, eventID)
	return err
}

// ---------- Transactional versions ----------

// SetOrderPagarmeOrderIDTx saves the Pagar.me order ID within a transaction.
func SetOrderPagarmeOrderIDTx(tx *sql.Tx, orderID, pagarmeOrderID string) error {
	_, err := tx.Exec(`UPDATE orders SET pagarme_order_id = ? WHERE id = ?`, pagarmeOrderID, orderID)
	return err
}

// SetOrderPagarmeChargeIDTx saves the Pagar.me charge ID within a transaction.
func SetOrderPagarmeChargeIDTx(tx *sql.Tx, orderID, chargeID string) error {
	_, err := tx.Exec(`UPDATE orders SET pagarme_charge_id = ? WHERE id = ?`, chargeID, orderID)
	return err
}

// PagarmeWebhookProcessedForOrder checks if we've already processed a webhook for this order+event_type combination.
// This prevents processing both order.paid and charge.paid for the same payment.
func PagarmeWebhookProcessedForOrder(db *sql.DB, orderID, eventType string) bool {
	var exists int
	query := `
		SELECT COUNT(*)
		FROM pagarme_webhook_events whe
		JOIN orders o ON o.pagarme_order_id = whe.pagarme_event_id
			OR o.pagarme_charge_id = whe.pagarme_event_id
		WHERE o.id = ?
			AND whe.event_type = ?
			AND whe.processed = 1
	`
	err := db.QueryRow(query, orderID, eventType).Scan(&exists)
	return err == nil && exists > 0
}

// MarkPagarmeWebhookEventProcessedAt marks a webhook event as processed with timestamp.
func MarkPagarmeWebhookEventProcessedAt(db *sql.DB, eventID string) error {
	_, err := db.Exec(
		`UPDATE pagarme_webhook_events SET processed = 1, processed_at = datetime('now') WHERE pagarme_event_id = ?`,
		eventID,
	)
	return err
}

// RecordOrderStatusChange logs an order status transition for audit purposes.
func RecordOrderStatusChange(tx *sql.Tx, orderID, oldStatus, newStatus, reason string, pagarmeEventID, pagarmeOrderID, pagarmeChargeID string) error {
	id := uuid.New().String()
	_, err := tx.Exec(
		`INSERT INTO order_status_history (id, order_id, old_status, new_status, reason, pagarme_event_id, pagarme_order_id, pagarme_charge_id) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		id, orderID, oldStatus, newStatus, reason, pagarmeEventID, pagarmeOrderID, pagarmeChargeID,
	)
	return err
}

// RecordOrderStatusChangeWithError logs a failed status transition attempt.
func RecordOrderStatusChangeWithError(tx *sql.Tx, orderID, oldStatus, newStatus, reason, errorMessage string) error {
	id := uuid.New().String()
	_, err := tx.Exec(
		`INSERT INTO order_status_history (id, order_id, old_status, new_status, reason, error_message) VALUES (?, ?, ?, ?, ?, ?)`,
		id, orderID, oldStatus, newStatus, reason, errorMessage,
	)
	return err
}
