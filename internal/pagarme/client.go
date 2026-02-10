// Package pagarme provides a Pagar.me V5 Core API client.
// Uses JSON API with Basic Auth. No external SDK dependency.
package pagarme

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const apiBaseURL = "https://api.pagar.me/core/v5"

// Client wraps all Pagar.me API interactions.
type Client struct {
	APIKey              string
	WebhookSecret       string
	PlatformRecipientID string // Pagar.me recipient ID for the Afterzin platform
	ApplicationFee      int64  // centavos per ticket (default 500 = R$5.00)
	BaseURL             string // platform frontend URL for redirects
	httpClient          *http.Client
}

// NewClient creates a Pagar.me client. Panics if apiKey is empty.
func NewClient(apiKey, webhookSecret, platformRecipientID string, applicationFee int64, baseURL string) *Client {
	if apiKey == "" {
		panic("PAGARME_API_KEY environment variable is required")
	}
	if applicationFee <= 0 {
		applicationFee = 500
	}
	return &Client{
		APIKey:              apiKey,
		WebhookSecret:       webhookSecret,
		PlatformRecipientID: platformRecipientID,
		ApplicationFee:      applicationFee,
		BaseURL:             strings.TrimRight(baseURL, "/"),
		httpClient:          &http.Client{},
	}
}

// authHeader returns the Basic Auth header value.
// Pagar.me V5: username = API key, password = empty string.
func (c *Client) authHeader() string {
	credentials := c.APIKey + ":"
	return "Basic " + base64.StdEncoding.EncodeToString([]byte(credentials))
}

// doRequest makes a JSON request to the Pagar.me V5 API.
func (c *Client) doRequest(method, path string, body interface{}) (map[string]interface{}, error) {
	url := apiBaseURL + path

	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal: %w", err)
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("new request: %w", err)
	}
	req.Header.Set("Authorization", c.authHeader())
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("pagarme error (%d): %s", resp.StatusCode, string(respBody))
	}

	var result map[string]interface{}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	return result, nil
}
