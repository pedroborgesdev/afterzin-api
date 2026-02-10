package qrcode

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

const separator = "."

// GenerateSignedPayload produces a signed payload for a ticket QR code.
// Format: ticketID + "." + hex(HMAC-SHA256(ticketID, secret)).
// The payload is unique, non-guessable and verifiable; it does not expose sensitive data in plain text.
func GenerateSignedPayload(ticketID string, secret []byte) string {
	if len(secret) == 0 {
		secret = []byte("default-secret")
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(ticketID))
	sig := mac.Sum(nil)
	return ticketID + separator + hex.EncodeToString(sig)
}

// VerifySignedPayload verifies the HMAC and returns the ticket ID if valid.
// Returns ("", false) if the payload is malformed or the signature is invalid.
func VerifySignedPayload(payload string, secret []byte) (ticketID string, ok bool) {
	idx := strings.LastIndex(payload, separator)
	if idx <= 0 || idx >= len(payload)-1 {
		return "", false
	}
	ticketID = payload[:idx]
	sigHex := payload[idx+1:]
	sig, err := hex.DecodeString(sigHex)
	if err != nil || len(sig) != sha256.Size {
		return "", false
	}
	if len(secret) == 0 {
		secret = []byte("default-secret")
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(ticketID))
	expected := mac.Sum(nil)
	return ticketID, hmac.Equal(sig, expected)
}

// GenerateSignedPayloadV2 creates a QR code payload containing ticket ID,
// charge ID, and event ID. This links the ticket to its payment transaction.
// Format: ticketID:chargeID:eventID.hmac_signature
func GenerateSignedPayloadV2(ticketID, chargeID, eventID string, secret []byte) string {
	if len(secret) == 0 {
		secret = []byte("default-secret")
	}
	data := ticketID + ":" + chargeID + ":" + eventID
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	sig := mac.Sum(nil)
	return data + separator + hex.EncodeToString(sig)
}

// VerifySignedPayloadV2 verifies a V2 QR code payload and extracts all components.
func VerifySignedPayloadV2(payload string, secret []byte) (ticketID, chargeID, eventID string, ok bool) {
	idx := strings.LastIndex(payload, separator)
	if idx <= 0 || idx >= len(payload)-1 {
		return "", "", "", false
	}
	data := payload[:idx]
	sigHex := payload[idx+1:]
	sig, err := hex.DecodeString(sigHex)
	if err != nil || len(sig) != sha256.Size {
		return "", "", "", false
	}
	if len(secret) == 0 {
		secret = []byte("default-secret")
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(data))
	expected := mac.Sum(nil)
	if !hmac.Equal(sig, expected) {
		return "", "", "", false
	}
	parts := strings.SplitN(data, ":", 3)
	if len(parts) != 3 {
		return "", "", "", false
	}
	return parts[0], parts[1], parts[2], true
}
