package application

import (
	"encoding/json"
	"fmt"
	"time"

	"bc_sandbox/internal/banks/domain"
	"bc_sandbox/internal/banks/ports"

	"github.com/gofrs/uuid"
)

type Service struct {
	repository ports.Repository
	clock      func() time.Time
}

type CreateBankCommand struct {
	Name   string
	Code   string
	Status string
}

type ChangeStatusCommand struct {
	BankID string
	Status string
}

func NewService(repository ports.Repository) Service {
	return Service{
		repository: repository,
		clock: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s Service) CreateBank(command CreateBankCommand) (domain.Bank, error) {
	bank, validation := domain.NewBank(command.Name, command.Code, command.Status, s.clock())
	if len(validation) > 0 {
		return domain.Bank{}, ValidationError{
			Code:    "validation_error",
			Message: "entrada invalida",
			Fields:  validation,
		}
	}

	if err := s.repository.Create(&bank); err != nil {
		return domain.Bank{}, fmt.Errorf("%w: %v", ErrBankPersistenceConflict, err)
	}

	if err := s.recordAuditEvent("bank", bank.ID, "bank.created", map[string]interface{}{
		"name":   bank.Name,
		"code":   bank.Code,
		"status": bank.Status,
	}); err != nil {
		return domain.Bank{}, err
	}

	return bank, nil
}

func (s Service) ListBanks() ([]domain.BankWithMetrics, error) {
	return s.repository.ListWithMetrics()
}

func (s Service) GetBank(rawID string) (domain.BankWithMetrics, error) {
	id, err := uuid.FromString(rawID)
	if err != nil {
		return domain.BankWithMetrics{}, ErrInvalidBankIdentifier
	}
	return s.repository.FindWithMetrics(id)
}

func (s Service) ChangeBankStatus(command ChangeStatusCommand) (domain.Bank, error) {
	id, err := uuid.FromString(command.BankID)
	if err != nil {
		return domain.Bank{}, ErrInvalidBankIdentifier
	}
	bank, err := s.repository.FindByID(id)
	if err != nil {
		return domain.Bank{}, err
	}

	previousStatus, validation := bank.ChangeStatus(command.Status, s.clock())
	if len(validation) > 0 {
		return domain.Bank{}, ValidationError{
			Code:    "validation_error",
			Message: "entrada invalida",
			Fields:  validation,
		}
	}

	if err := s.repository.Update(&bank); err != nil {
		return domain.Bank{}, fmt.Errorf("%w: %v", ErrBankPersistenceConflict, err)
	}
	if err := s.recordAuditEvent("bank", bank.ID, "bank.status_changed", map[string]interface{}{
		"from": previousStatus,
		"to":   bank.Status,
	}); err != nil {
		return domain.Bank{}, err
	}

	return bank, nil
}

func (s Service) EnsureBankCanInitiateTransactions(bankID uuid.UUID) error {
	bank, err := s.repository.FindByID(bankID)
	if err != nil {
		return err
	}
	if !bank.CanInitiateTransactions() {
		return ErrBankCannotInitiate
	}
	return nil
}

func (s Service) recordAuditEvent(entityType string, entityID uuid.UUID, eventType string, payload map[string]interface{}) error {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	event := domain.AuditEvent{
		ID:         uuid.Must(uuid.NewV4()),
		EntityType: entityType,
		EntityID:   entityID,
		EventType:  eventType,
		Payload:    string(payloadBytes),
		CreatedAt:  s.clock(),
	}
	if err := s.repository.CreateAuditEvent(&event); err != nil {
		return fmt.Errorf("%w: %v", ErrAuditPersistenceConflict, err)
	}
	return nil
}
