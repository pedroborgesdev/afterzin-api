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
