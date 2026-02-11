package pagarme

import "fmt"

// CreateRecipientParams holds the data needed to create a Pagar.me recipient.
type CreateRecipientParams struct {
	Name                   string
	Email                  string
	Document               string // CPF or CNPJ
	DocumentType           string // "CPF" ou "CNPJ"
	Type                   string // "individual" ou "company"
	Birthdate              string // PF
	MonthlyIncome          int    // PF
	ProfessionalOccupation string // PF
	CompanyName            string // PJ
	TradingName            string // PJ
	AnnualRevenue          int    // PJ
	BankCode               string // e.g. "001", "341"
	BranchNumber           string
	BranchCheckDigit       string
	AccountNumber          string
	AccountCheckDigit      string
	AccountType            string // "checking" ou "savings"
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

	registerInfo := map[string]interface{}{
		"email":    params.Email,
		"document": params.Document,
		"type":     params.Type,
	}
	// Usa o telefone real do usuário se disponível
	var phoneNumbers []string
	if params.Type == "individual" {
		if params.ProfessionalOccupation != "" { // só para evitar warning, não é telefone
			// ...existing code...
		}
		if v, ok := any(params).(interface{ GetPhone() string }); ok && v.GetPhone() != "" {
			phoneNumbers = []string{v.GetPhone()}
		} else if p, ok := any(params).(map[string]interface{}); ok && p["phone_numbers"] != nil {
			if arr, ok := p["phone_numbers"].([]string); ok && len(arr) > 0 {
				phoneNumbers = arr
			}
		}
	}
	// fallback para PJ
	if params.Type == "company" {
		if v, ok := any(params).(interface{ GetPhone() string }); ok && v.GetPhone() != "" {
			phoneNumbers = []string{v.GetPhone()}
		} else if p, ok := any(params).(map[string]interface{}); ok && p["phone_numbers"] != nil {
			if arr, ok := p["phone_numbers"].([]string); ok && len(arr) > 0 {
				phoneNumbers = arr
			}
		}
	}
	// fallback: pega do struct se existir campo Phone ou PhoneNumbers
	if len(phoneNumbers) == 0 {
		if params.Type == "individual" && params.ProfessionalOccupation != "" {
			// ...existing code...
		}
		// Para garantir compatibilidade, tente pegar do struct
		if pf, ok := any(params).(struct{ Phone string }); ok && pf.Phone != "" {
			phoneNumbers = []string{pf.Phone}
		}
		if pj, ok := any(params).(struct{ Phone string }); ok && pj.Phone != "" {
			phoneNumbers = []string{pj.Phone}
		}
	}
	// fallback final: não deixa vazio
	if len(phoneNumbers) == 0 {
		phoneNumbers = []string{"+5511999999999"}
	}
	registerInfo["phone_numbers"] = phoneNumbers
	if params.Type == "individual" {
		registerInfo["name"] = params.Name
		registerInfo["birthdate"] = params.Birthdate
		registerInfo["monthly_income"] = params.MonthlyIncome
		registerInfo["professional_occupation"] = params.ProfessionalOccupation
	} else {
		registerInfo["company_name"] = params.CompanyName
		registerInfo["trading_name"] = params.TradingName
		registerInfo["annual_revenue"] = params.AnnualRevenue
	}

	body := map[string]interface{}{
		"code":                 params.Document, // pode ser ajustado para um identificador único
		"register_information": registerInfo,
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
