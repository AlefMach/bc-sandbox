package ports

import (
	"bc_sandbox/internal/banks/domain"

	"github.com/gofrs/uuid"
)

type Repository interface {
	Create(bank *domain.Bank) error
	FindByID(id uuid.UUID) (domain.Bank, error)
	ListWithMetrics() ([]domain.BankWithMetrics, error)
	FindWithMetrics(id uuid.UUID) (domain.BankWithMetrics, error)
	Update(bank *domain.Bank) error
	CreateAuditEvent(event *domain.AuditEvent) error
}
