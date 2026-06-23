package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

const (
	KeyTypeCPF    = "cpf"
	KeyTypeCNPJ   = "cnpj"
	KeyTypeEmail  = "email"
	KeyTypePhone  = "phone"
	KeyTypeRandom = "random"

	KeyStatusActive   = "active"
	KeyStatusInactive = "inactive"
)

var (
	validKeyTypes = map[string]bool{
		KeyTypeCPF:    true,
		KeyTypeCNPJ:   true,
		KeyTypeEmail:  true,
		KeyTypePhone:  true,
		KeyTypeRandom: true,
	}
	validKeyStatuses = map[string]bool{
		KeyStatusActive:   true,
		KeyStatusInactive: true,
	}
	pixEmailPattern  = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)
	randomKeyPattern = regexp.MustCompile(`^[A-Za-z0-9._\-]{8,77}$`)
)

type PixKey struct {
	ID        uuid.UUID `db:"id" json:"id"`
	AccountID uuid.UUID `db:"account_id" json:"account_id"`
	KeyType   string    `db:"key_type" json:"key_type"`
	KeyValue  string    `db:"key_value" json:"key"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type PixKeys []PixKey

type LookupResult struct {
	BankID         uuid.UUID `json:"bank_id"`
	BankName       string    `json:"bank_name"`
	BankCode       string    `json:"bank_code"`
	AccountID      uuid.UUID `json:"account_id"`
	Agency         string    `json:"agency"`
	Number         string    `json:"number"`
	AccountStatus  string    `json:"account_status"`
	HolderID       uuid.UUID `json:"holder_id"`
	HolderName     string    `json:"holder_name"`
	HolderDocument string    `json:"holder_document"`
	KeyType        string    `json:"key_type"`
	Key            string    `json:"key"`
	KeyStatus      string    `json:"status"`
}

func NewPixKey(accountID uuid.UUID, keyType string, value string, now time.Time) (PixKey, map[string]string) {
	keyType = NormalizeKeyType(keyType)
	keyValue := NormalizeKeyValue(keyType, value)

	if validation := ValidatePixKeyInput(keyType, keyValue); len(validation) > 0 {
		return PixKey{}, validation
	}

	return PixKey{
		ID:        uuid.Must(uuid.NewV4()),
		AccountID: accountID,
		KeyType:   keyType,
		KeyValue:  keyValue,
		Status:    KeyStatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func NormalizeKeyType(keyType string) string {
	keyType = strings.TrimSpace(strings.ToLower(keyType))
	switch keyType {
	case "cpf":
		return KeyTypeCPF
	case "cnpj":
		return KeyTypeCNPJ
	case "email", "e-mail":
		return KeyTypeEmail
	case "telefone", "phone", "celular":
		return KeyTypePhone
	case "aleatoria", "aleatória", "random", "uuid":
		return KeyTypeRandom
	default:
		return keyType
	}
}

func NormalizeLookupKey(value string) string {
	value = strings.TrimSpace(value)
	if pixEmailPattern.MatchString(value) {
		return strings.ToLower(value)
	}
	digits := onlyDigits(value)
	if len(digits) == 11 || len(digits) == 14 || (len(digits) >= 10 && len(digits) <= 15) {
		return digits
	}
	return value
}

func NormalizeKeyValue(keyType string, value string) string {
	value = strings.TrimSpace(value)
	switch keyType {
	case KeyTypeCPF, KeyTypeCNPJ, KeyTypePhone:
		return onlyDigits(value)
	case KeyTypeEmail:
		return strings.ToLower(value)
	default:
		return value
	}
}

func ValidatePixKeyInput(keyType string, value string) map[string]string {
	errors := map[string]string{}
	if !validKeyTypes[keyType] {
		errors["key_type"] = fmt.Sprintf("tipo de chave deve ser um de: %s, %s, %s, %s, %s", KeyTypeCPF, KeyTypeCNPJ, KeyTypeEmail, KeyTypePhone, KeyTypeRandom)
		return errors
	}
	if value == "" {
		errors["key"] = "chave Pix e obrigatoria"
		return errors
	}

	switch keyType {
	case KeyTypeCPF:
		if len(value) != 11 {
			errors["key"] = "CPF deve possuir 11 digitos"
		}
	case KeyTypeCNPJ:
		if len(value) != 14 {
			errors["key"] = "CNPJ deve possuir 14 digitos"
		}
	case KeyTypeEmail:
		if !pixEmailPattern.MatchString(value) || len(value) > 254 {
			errors["key"] = "e-mail invalido"
		}
	case KeyTypePhone:
		if len(value) < 10 || len(value) > 15 {
			errors["key"] = "telefone deve possuir de 10 a 15 digitos"
		}
	case KeyTypeRandom:
		if len(value) > 254 || !isValidRandomKey(value) {
			errors["key"] = "chave aleatoria deve ser um UUID ou conter de 8 a 77 caracteres validos"
		}
	}
	return errors
}

func KeyTypeLabel(keyType string) string {
	switch keyType {
	case KeyTypeCPF:
		return "CPF"
	case KeyTypeCNPJ:
		return "CNPJ"
	case KeyTypeEmail:
		return "E-mail"
	case KeyTypePhone:
		return "Telefone"
	case KeyTypeRandom:
		return "Chave aleatoria"
	default:
		return keyType
	}
}

func KeyStatusLabel(status string) string {
	switch status {
	case KeyStatusActive:
		return "ativa"
	case KeyStatusInactive:
		return "inativa"
	default:
		return status
	}
}

func ValidKeyStatus(status string) bool {
	return validKeyStatuses[strings.TrimSpace(strings.ToLower(status))]
}

func isValidRandomKey(value string) bool {
	if _, err := uuid.FromString(value); err == nil {
		return true
	}
	return randomKeyPattern.MatchString(value)
}

func onlyDigits(value string) string {
	var builder strings.Builder
	for _, char := range value {
		if char >= '0' && char <= '9' {
			builder.WriteRune(char)
		}
	}
	return builder.String()
}
