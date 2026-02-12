package pagarme

import (
	"fmt"
	"regexp"
	"strconv"
)

// PhoneData representa telefone estruturado para envio ao gateway de pagamento
type PhoneData struct {
	CountryCode string // "55" para Brasil
	AreaCode    string // DDD (2 dígitos)
	Number      string // Número (8 ou 9 dígitos)
}

// sanitizePhone remove caracteres não numéricos de um telefone
func sanitizePhone(phone string) string {
	return regexp.MustCompile(`[^\d]`).ReplaceAllString(phone, "")
}

// ValidatePhone valida telefone completo com suporte a Brasil e internacional
func ValidatePhone(countryCode, areaCode, number string) error {
	// Valida country code não vazio
	if countryCode == "" {
		return fmt.Errorf("country code é obrigatório")
	}

	// Sanitiza todos os campos
	cc := sanitizePhone(countryCode)
	ac := sanitizePhone(areaCode)
	num := sanitizePhone(number)

	// Validação específica para Brasil (55)
	if cc == "55" {
		// Area code (DDD): deve ter 2 dígitos
		if len(ac) != 2 {
			return fmt.Errorf("DDD deve ter 2 dígitos")
		}

		// Valida se DDD está na faixa válida (11-99)
		ddd, err := strconv.Atoi(ac)
		if err != nil || ddd < 11 || ddd > 99 {
			return fmt.Errorf("DDD inválido: deve estar entre 11 e 99")
		}

		// Número: deve ter 8 ou 9 dígitos
		numLen := len(num)
		if numLen != 8 && numLen != 9 {
			return fmt.Errorf("número deve ter 8 ou 9 dígitos (recebido %d)", numLen)
		}
	} else {
		// Validação internacional: mais flexível
		if len(cc) < 1 || len(cc) > 3 {
			return fmt.Errorf("country code inválido")
		}

		if len(ac) == 0 {
			return fmt.Errorf("area code é obrigatório")
		}

		if len(num) < 6 {
			return fmt.Errorf("número de telefone muito curto")
		}
	}

	return nil
}

// ParsePhone converte telefone para estrutura sanitizada do Pagar.me
func ParsePhone(countryCode, areaCode, number string) PhoneData {
	return PhoneData{
		CountryCode: sanitizePhone(countryCode),
		AreaCode:    sanitizePhone(areaCode),
		Number:      sanitizePhone(number),
	}
}
