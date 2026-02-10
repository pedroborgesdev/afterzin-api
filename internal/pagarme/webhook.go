package pagarme

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// WebhookEvent represents a parsed Pagar.me webhook event.
type WebhookEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	CreatedAt string                 `json:"created_at"`
	Data      map[string]interface{} `json:"data"`
}

// VerifyWebhookSignature verifies the x-hub-signature header against the payload.
//
// Pagar.me V5 webhook verification:
//  1. Get the x-hub-signature header (format: "sha256=<hex_hmac>")
//  2. Compute HMAC-SHA256 of the raw body using the webhook secret
//  3. Compare the computed signature with the provided one
func (c *Client) VerifyWebhookSignature(payload []byte, signatureHeader string) (*WebhookEvent, error) {
	if c.WebhookSecret == "" {
		return nil, fmt.Errorf("webhook secret not configured")
	}

	// Header format: sha256=HEXHASH
	if !strings.HasPrefix(signatureHeader, "sha256=") {
		return nil, fmt.Errorf("invalid signature format: expected sha256= prefix")
	}
	providedSig := strings.TrimPrefix(signatureHeader, "sha256=")

	// Compute expected HMAC-SHA256
	mac := hmac.New(sha256.New, []byte(c.WebhookSecret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(providedSig), []byte(expectedSig)) {
		return nil, fmt.Errorf("signature verification failed")
	}

	// Parse the event body
	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return nil, fmt.Errorf("parse event: %w", err)
	}

	return &event, nil
}
