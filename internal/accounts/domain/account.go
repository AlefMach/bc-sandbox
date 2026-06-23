package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

const (
	AccountStatusActive  = "active"
	AccountStatusBlocked = "blocked"
	AccountStatusClosed  = "closed"
)

var (
	validAccountStatuses = map[string]bool{
		AccountStatusActive:  true,
		AccountStatusBlocked: true,
		AccountStatusClosed:  true,
	}
	agencyPattern        = regexp.MustCompile(`^[0-9]{1,8}$`)
	accountNumberPattern = regexp.MustCompile(`^[0-9]{1,20}$`)
)

type Account struct {
	ID           uuid.UUID `db:"id" json:"id"`
	BankID       uuid.UUID `db:"bank_id" json:"bank_id"`
	CustomerID   uuid.UUID `db:"customer_id" json:"customer_id"`
	Agency       string    `db:"agency" json:"agency"`
	Number       string    `db:"number" json:"number"`
	BalanceCents int64     `db:"balance_cents" json:"balance_cents"`
	Status       string    `db:"status" json:"status"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
	UpdatedAt    time.Time `db:"updated_at" json:"updated_at"`
}

type Accounts []Account

type Balance struct {
	AccountID     uuid.UUID `json:"account_id"`
	BankID        uuid.UUID `json:"bank_id"`
	BankName      string    `json:"bank_name"`
	CustomerID    uuid.UUID `json:"customer_id"`
	CustomerName  string    `json:"customer_name"`
	Agency        string    `json:"agency"`
	Number        string    `json:"number"`
	BalanceCents  int64     `json:"balance_cents"`
	BalanceAmount string    `json:"balance_amount"`
	Status        string    `json:"status"`
}

func NewAccount(bankID uuid.UUID, customerID uuid.UUID, agency string, number string, balanceCents int64, status string, now time.Time) (Account, map[string]string) {
	agency = onlyDigits(agency)
	number = onlyDigits(number)
	status = NormalizeAccountStatus(status)
	if status == "" {
		status = AccountStatusActive
	}

	if validation := ValidateAccountInput(agency, number, balanceCents, status); len(validation) > 0 {
		return Account{}, validation
	}

	return Account{
		ID:           uuid.Must(uuid.NewV4()),
		BankID:       bankID,
		CustomerID:   customerID,
		Agency:       agency,
		Number:       number,
		BalanceCents: balanceCents,
		Status:       status,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func NormalizeAccountStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	switch status {
	case "ativa", "ativo":
		return AccountStatusActive
	case "bloqueada", "bloqueado":
		return AccountStatusBlocked
	case "encerrada", "encerrado":
		return AccountStatusClosed
	default:
		return status
	}
}

func AccountStatusLabel(status string) string {
	switch status {
	case AccountStatusActive:
		return "ativa"
	case AccountStatusBlocked:
		return "bloqueada"
	case AccountStatusClosed:
		return "encerrada"
	default:
		return status
	}
}

func ValidAccountStatus(status string) bool {
	return validAccountStatuses[NormalizeAccountStatus(status)]
}

func ValidateAccountInput(agency string, number string, balanceCents int64, status string) map[string]string {
	errors := map[string]string{}
	if !agencyPattern.MatchString(agency) {
		errors["agency"] = "agencia deve possuir de 1 a 8 digitos"
	}
	if !accountNumberPattern.MatchString(number) {
		errors["number"] = "numero da conta deve possuir de 1 a 20 digitos"
	}
	if balanceCents < 0 {
		errors["balance_cents"] = "saldo nao pode ser negativo"
	}
	if status != "" && !ValidAccountStatus(status) {
		errors["status"] = fmt.Sprintf("status deve ser um de: %s, %s, %s", AccountStatusActive, AccountStatusBlocked, AccountStatusClosed)
	}
	return errors
}

func (a Account) CanOperate() bool {
	return a.Status == AccountStatusActive
}
