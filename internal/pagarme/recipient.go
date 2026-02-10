package pagarme

import "fmt"

// CreateRecipientParams holds the data needed to create a Pagar.me recipient.
type CreateRecipientParams struct {
	Name              string
	Email             string
	Document          string // CPF or CNPJ
	DocumentType      string // "CPF" or "CNPJ"
	Type              string // "individual" or "company"
	BankCode          string // e.g. "001", "341"
	BranchNumber      string
	BranchCheckDigit  string
	AccountNumber     string
	AccountCheckDigit string
	AccountType       string // "checking" or "savings"
}

// RecipientResult contains the recipient data returned after creation.
type RecipientResult struct {
	RecipientID string `json:"recipientId"`
	Status      string `json:"status"`
	Name        string `json:"name"`
}

// CreateRecipient creates a new recipient in Pagar.me.
//
// A recipient represents a producer who can receive split payments.
// The default bank account is used for automatic transfers.
func (c *Client) CreateRecipient(params CreateRecipientParams) (*RecipientResult, error) {
	holderType := "individual"
	if params.Type == "company" {
		holderType = "company"
	}

	body := map[string]interface{}{
		"name":     params.Name,
		"email":    params.Email,
		"document": params.Document,
		"type":     params.Type,
		"default_bank_account": map[string]interface{}{
			"holder_name":         params.Name,
			"holder_type":         holderType,
			"holder_document":     params.Document,
			"bank":                params.BankCode,
			"branch_number":       params.BranchNumber,
			"branch_check_digit":  params.BranchCheckDigit,
			"account_number":      params.AccountNumber,
			"account_check_digit": params.AccountCheckDigit,
			"type":                params.AccountType,
		},
		"transfer_settings": map[string]interface{}{
			"transfer_enabled":  true,
			"transfer_interval": "daily",
			"transfer_day":      0,
		},
	}

	result, err := c.doRequest("POST", "/recipients", body)
	if err != nil {
		return nil, fmt.Errorf("create recipient: %w", err)
	}

	id, _ := result["id"].(string)
	status, _ := result["status"].(string)
	name, _ := result["name"].(string)

	if id == "" {
		return nil, fmt.Errorf("no recipient id in response")
	}

	return &RecipientResult{
		RecipientID: id,
		Status:      status,
		Name:        name,
	}, nil
}

// GetRecipient retrieves a recipient's details from Pagar.me.
func (c *Client) GetRecipient(recipientID string) (map[string]interface{}, error) {
	return c.doRequest("GET", "/recipients/"+recipientID, nil)
}
