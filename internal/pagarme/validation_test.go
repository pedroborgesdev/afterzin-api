package pagarme

import (
	"testing"
)

func TestValidatePaymentMethod(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		wantErr bool
	}{
		{
			name:    "pix válido",
			method:  "pix",
			wantErr: false,
		},
		{
			name:    "credit_card inválido",
			method:  "credit_card",
			wantErr: true,
		},
		{
			name:    "boleto inválido",
			method:  "boleto",
			wantErr: true,
		},
		{
			name:    "voucher inválido",
			method:  "voucher",
			wantErr: true,
		},
		{
			name:    "vazio inválido",
			method:  "",
			wantErr: true,
		},
		{
			name:    "método desconhecido inválido",
			method:  "unknown",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePaymentMethod(tt.method)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePaymentMethod() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeDocument(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "CPF com pontos e hífen",
			input:    "123.456.789-00",
			expected: "12345678900",
		},
		{
			name:     "CPF já limpo",
			input:    "12345678900",
			expected: "12345678900",
		},
		{
			name:     "CPF com hífens múltiplos",
			input:    "123-456-789-00",
			expected: "12345678900",
		},
		{
			name:     "CPF vazio",
			input:    "",
			expected: "",
		},
		{
			name:     "CPF com espaços",
			input:    "123 456 789 00",
			expected: "12345678900",
		},
		{
			name:     "CNPJ com formatação",
			input:    "12.345.678/0001-90",
			expected: "12345678000190",
		},
		{
			name:     "Apenas caracteres não numéricos",
			input:    "abc-def.ghi/jkl",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeDocument(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeDocument() = %v, want %v", result, tt.expected)
			}
		})
	}
}
