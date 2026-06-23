package domain

import (
	"regexp"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

var (
	documentPattern = regexp.MustCompile(`^[0-9]{11}$|^[0-9]{14}$`)
	emailPattern    = regexp.MustCompile(`^[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}$`)
)

type Customer struct {
	ID        uuid.UUID `db:"id" json:"id"`
	BankID    uuid.UUID `db:"bank_id" json:"bank_id"`
	Name      string    `db:"name" json:"name"`
	Document  string    `db:"document" json:"document"`
	Email     string    `db:"email" json:"email"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type Customers []Customer

func NewCustomer(bankID uuid.UUID, name string, document string, email string, now time.Time) (Customer, map[string]string) {
	name = strings.TrimSpace(name)
	document = onlyDigits(document)
	email = strings.TrimSpace(strings.ToLower(email))

	if validation := ValidateCustomerInput(name, document, email); len(validation) > 0 {
		return Customer{}, validation
	}

	return Customer{
		ID:        uuid.Must(uuid.NewV4()),
		BankID:    bankID,
		Name:      name,
		Document:  document,
		Email:     email,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func ValidateCustomerInput(name string, document string, email string) map[string]string {
	errors := map[string]string{}
	if name == "" {
		errors["name"] = "nome do cliente e obrigatorio"
	}
	if len(name) > 160 {
		errors["name"] = "nome do cliente deve ter no maximo 160 caracteres"
	}
	if !documentPattern.MatchString(document) {
		errors["document"] = "documento deve possuir 11 ou 14 digitos"
	}
	if !emailPattern.MatchString(email) || len(email) > 254 {
		errors["email"] = "e-mail invalido"
	}
	return errors
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
