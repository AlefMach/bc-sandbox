package domain

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/gofrs/uuid"
)

const (
	StatusActive      = "active"
	StatusInactive    = "inactive"
	StatusOffline     = "offline"
	StatusMaintenance = "maintenance"
)

var validStatuses = map[string]bool{
	StatusActive:      true,
	StatusInactive:    true,
	StatusOffline:     true,
	StatusMaintenance: true,
}

var bankCodePattern = regexp.MustCompile(`^[0-9]{3}$`)

type Bank struct {
	ID        uuid.UUID `db:"id" json:"id"`
	Name      string    `db:"name" json:"name"`
	Code      string    `db:"code" json:"code"`
	Status    string    `db:"status" json:"status"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
	UpdatedAt time.Time `db:"updated_at" json:"updated_at"`
}

type Banks []Bank

type Metrics struct {
	Accounts              int64 `json:"accounts"`
	TotalTransactedCents  int64 `json:"total_transacted_cents"`
	PendingTransactions   int64 `json:"pending_transactions"`
	CompletedTransactions int64 `json:"completed_transactions"`
	FailedTransactions    int64 `json:"failed_transactions"`
}

type BankWithMetrics struct {
	Bank
	Metrics Metrics `json:"metrics"`
}

type AuditEvent struct {
	ID         uuid.UUID `db:"id" json:"id"`
	EntityType string    `db:"entity_type" json:"entity_type"`
	EntityID   uuid.UUID `db:"entity_id" json:"entity_id"`
	EventType  string    `db:"event_type" json:"event_type"`
	Payload    string    `db:"payload" json:"payload"`
	CreatedAt  time.Time `db:"created_at" json:"created_at"`
}

func NewBank(name string, code string, status string, now time.Time) (Bank, map[string]string) {
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	status = NormalizeStatus(status)
	if status == "" {
		status = StatusActive
	}
	if validation := ValidateBankInput(name, code, status); len(validation) > 0 {
		return Bank{}, validation
	}

	return Bank{
		ID:        uuid.Must(uuid.NewV4()),
		Name:      name,
		Code:      code,
		Status:    status,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func NormalizeStatus(status string) string {
	status = strings.TrimSpace(strings.ToLower(status))
	switch status {
	case "ativo":
		return StatusActive
	case "inativo":
		return StatusInactive
	case "em manutencao", "em manutenção":
		return StatusMaintenance
	default:
		return status
	}
}

func StatusLabel(status string) string {
	switch status {
	case StatusActive:
		return "ativo"
	case StatusInactive:
		return "inativo"
	case StatusOffline:
		return "offline"
	case StatusMaintenance:
		return "em manutencao"
	default:
		return status
	}
}

func ValidStatus(status string) bool {
	return validStatuses[NormalizeStatus(status)]
}

func ValidateBankInput(name string, code string, status string) map[string]string {
	errors := map[string]string{}
	name = strings.TrimSpace(name)
	code = strings.TrimSpace(code)
	status = NormalizeStatus(status)

	if name == "" {
		errors["name"] = "nome do banco e obrigatorio"
	}
	if len(name) > 160 {
		errors["name"] = "nome do banco deve ter no maximo 160 caracteres"
	}
	if !bankCodePattern.MatchString(code) {
		errors["code"] = "codigo bancario deve possuir exatamente 3 digitos"
	}
	if status != "" && !ValidStatus(status) {
		errors["status"] = fmt.Sprintf("status deve ser um de: %s, %s, %s, %s", StatusActive, StatusInactive, StatusOffline, StatusMaintenance)
	}

	return errors
}

func ValidateStatus(status string) map[string]string {
	errors := map[string]string{}
	if !ValidStatus(status) {
		errors["status"] = fmt.Sprintf("status deve ser um de: %s, %s, %s, %s", StatusActive, StatusInactive, StatusOffline, StatusMaintenance)
	}
	return errors
}

func (b Bank) CanInitiateTransactions() bool {
	return b.Status != StatusInactive && b.Status != StatusOffline
}

func (b *Bank) ChangeStatus(status string, now time.Time) (string, map[string]string) {
	status = NormalizeStatus(status)
	if validation := ValidateStatus(status); len(validation) > 0 {
		return b.Status, validation
	}
	previousStatus := b.Status
	b.Status = status
	b.UpdatedAt = now
	return previousStatus, nil
}
