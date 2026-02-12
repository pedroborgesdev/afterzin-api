package pagarme

import "fmt"

const (
	// AllowedPaymentMethod define o único método de pagamento permitido na plataforma
	AllowedPaymentMethod = "pix"

	// PixExpirationSeconds define o tempo de expiração do PIX em segundos (15 minutos)
	PixExpirationSeconds = 900

	// AllowedDocumentType define o tipo de documento aceito
	AllowedDocumentType = "CPF"

	// AllowedCustomerType define o tipo de cliente aceito
	AllowedCustomerType = "individual"
)

// ValidatePaymentMethod verifica se o método de pagamento fornecido é válido.
// Retorna erro se o método não for "pix".
func ValidatePaymentMethod(method string) error {
	if method == "" {
		return fmt.Errorf("método de pagamento não pode estar vazio")
	}
	if method != AllowedPaymentMethod {
		return fmt.Errorf("método de pagamento inválido: apenas '%s' é permitido, recebido '%s'", AllowedPaymentMethod, method)
	}
	return nil
}
