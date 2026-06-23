package domain

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

const (
	TransactionTypePix = "pix"

	StatusCreated                   = "created"
	StatusPayerAccountValidated     = "payer_account_validated"
	StatusPixKeyConsulted           = "pix_key_consulted"
	StatusReceiverAccountIdentified = "receiver_account_identified"
	StatusBalanceValidated          = "balance_validated"
	StatusFundsReserved             = "funds_reserved"
	StatusSentToCentralBank         = "sent_to_central_bank"
	StatusSettled                   = "settled"
	StatusReceiverBankNotified      = "receiver_bank_notified"
	StatusReceiverAccountCredited   = "receiver_account_credited"
	StatusCompleted                 = "completed"
	StatusFailed                    = "failed"

	ServiceOriginBank     = "origin_bank"
	ServiceCentralBank    = "central_bank"
	ServiceReceiverBank   = "receiver_bank"
	ServiceStateMachine   = "state_machine"
	EventTypeTransition   = "status_transition"
	EventTypeInvalidState = "invalid_transition"
)

var allowedTransitions = map[string]map[string]bool{
	StatusCreated: {
		StatusPayerAccountValidated: true,
		StatusFailed:                true,
	},
	StatusPayerAccountValidated: {
		StatusPixKeyConsulted: true,
		StatusFailed:          true,
	},
	StatusPixKeyConsulted: {
		StatusReceiverAccountIdentified: true,
		StatusFailed:                    true,
	},
	StatusReceiverAccountIdentified: {
		StatusBalanceValidated: true,
		StatusFailed:           true,
	},
	StatusBalanceValidated: {
		StatusFundsReserved: true,
		StatusFailed:        true,
	},
	StatusFundsReserved: {
		StatusSentToCentralBank: true,
		StatusFailed:            true,
	},
	StatusSentToCentralBank: {
		StatusSettled: true,
		StatusFailed:  true,
	},
	StatusSettled: {
		StatusReceiverBankNotified: true,
		StatusFailed:               true,
	},
	StatusReceiverBankNotified: {
		StatusReceiverAccountCredited: true,
		StatusFailed:                  true,
	},
	StatusReceiverAccountCredited: {
		StatusCompleted: true,
		StatusFailed:    true,
	},
}

type Transaction struct {
	ID                uuid.UUID  `db:"id" json:"id"`
	Type              string     `db:"transaction_type" json:"type"`
	Status            string     `db:"status" json:"status"`
	AmountCents       int64      `db:"amount_cents" json:"amount_cents"`
	PayerBankID       uuid.UUID  `db:"payer_bank_id" json:"payer_bank_id"`
	PayerAccountID    uuid.UUID  `db:"payer_account_id" json:"payer_account_id"`
	ReceiverBankID    uuid.UUID  `db:"receiver_bank_id" json:"receiver_bank_id"`
	ReceiverAccountID uuid.UUID  `db:"receiver_account_id" json:"receiver_account_id"`
	PixKey            string     `db:"pix_key" json:"pix_key"`
	ReservedAt        *time.Time `db:"reserved_at" json:"reserved_at,omitempty"`
	SettledAt         *time.Time `db:"settled_at" json:"settled_at,omitempty"`
	CreditedAt        *time.Time `db:"credited_at" json:"credited_at,omitempty"`
	FailureReason     string     `db:"failure_reason" json:"failure_reason,omitempty"`
	CreatedAt         time.Time  `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time  `db:"updated_at" json:"updated_at"`
}

type Transactions []Transaction

func (Transaction) TableName() string {
	return "pix_transactions"
}

type TransactionEvent struct {
	ID            uuid.UUID `db:"id" json:"id"`
	TransactionID uuid.UUID `db:"transaction_id" json:"transaction_id"`
	Type          string    `db:"transaction_type" json:"type"`
	EventType     string    `db:"event_type" json:"event_type"`
	Previous      string    `db:"previous_status" json:"previous_status"`
	Next          string    `db:"new_status" json:"new_status"`
	Message       string    `db:"message" json:"message"`
	Service       string    `db:"service" json:"service"`
	BankID        uuid.UUID `db:"bank_id" json:"bank_id"`
	Metadata      string    `db:"metadata" json:"metadata"`
	CreatedAt     time.Time `db:"created_at" json:"created_at"`
}

type TransactionEvents []TransactionEvent

func (TransactionEvent) TableName() string {
	return "pix_transaction_events"
}

type TimelineItem struct {
	Time        time.Time `json:"time"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	Service     string    `json:"service"`
	BankID      uuid.UUID `json:"bank_id"`
	Metadata    string    `json:"metadata"`
}

func NewTransaction(payerAccountID uuid.UUID, amountCents int64, pixKey string, now time.Time) (Transaction, map[string]string) {
	pixKey = NormalizeLookupKey(pixKey)
	if validation := ValidateTransactionInput(amountCents, pixKey); len(validation) > 0 {
		return Transaction{}, validation
	}
	return Transaction{
		ID:             uuid.Must(uuid.NewV4()),
		Type:           TransactionTypePix,
		Status:         StatusCreated,
		AmountCents:    amountCents,
		PayerAccountID: payerAccountID,
		PixKey:         pixKey,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func ValidateTransactionInput(amountCents int64, pixKey string) map[string]string {
	errors := map[string]string{}
	if amountCents <= 0 {
		errors["amount_cents"] = "valor deve ser maior que zero"
	}
	if strings.TrimSpace(pixKey) == "" {
		errors["pix_key"] = "chave Pix e obrigatoria"
	}
	return errors
}

func CanTransition(from string, to string) bool {
	return allowedTransitions[from][to]
}

func StatusLabel(status string) string {
	switch status {
	case StatusCreated:
		return "transacao criada"
	case StatusPayerAccountValidated:
		return "conta pagadora validada"
	case StatusPixKeyConsulted:
		return "chave Pix consultada"
	case StatusReceiverAccountIdentified:
		return "conta recebedora identificada"
	case StatusBalanceValidated:
		return "saldo validado"
	case StatusFundsReserved:
		return "fundos reservados"
	case StatusSentToCentralBank:
		return "solicitacao enviada ao Banco Central"
	case StatusSettled:
		return "transacao liquidada"
	case StatusReceiverBankNotified:
		return "banco recebedor notificado"
	case StatusReceiverAccountCredited:
		return "conta recebedora creditada"
	case StatusCompleted:
		return "transacao concluida"
	case StatusFailed:
		return "transacao falhou"
	default:
		return status
	}
}

func TransitionMessage(status string) string {
	return fmt.Sprintf("Pix: %s", StatusLabel(status))
}
