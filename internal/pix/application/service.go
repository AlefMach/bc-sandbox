package application

import (
	"errors"
	"fmt"
	"time"

	accountdomain "bc_sandbox/internal/accounts/domain"
	"bc_sandbox/internal/pix/domain"
	"bc_sandbox/internal/pix/ports"

	"github.com/gofrs/uuid"
)

type Service struct {
	repository ports.Repository
	clock      func() time.Time
}

type CreatePixKeyCommand struct {
	AccountID string
	KeyType   string
	Key       string
}

func NewService(repository ports.Repository) Service {
	return Service{
		repository: repository,
		clock: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s Service) CreatePixKey(command CreatePixKeyCommand) (domain.PixKey, error) {
	accountID, err := uuid.FromString(command.AccountID)
	if err != nil {
		return domain.PixKey{}, ErrInvalidAccountIdentifier
	}

	account, err := s.repository.FindAccountByID(accountID)
	if err != nil {
		return domain.PixKey{}, err
	}
	if account.Status != accountdomain.AccountStatusActive {
		return domain.PixKey{}, ErrAccountCannotReceivePixKey
	}

	pixKey, validation := domain.NewPixKey(accountID, command.KeyType, command.Key, s.clock())
	if len(validation) > 0 {
		return domain.PixKey{}, ValidationError{Code: "validation_error", Message: "entrada invalida", Fields: validation}
	}

	if _, err := s.repository.FindPixKey(pixKey.KeyValue); err == nil {
		return domain.PixKey{}, ErrPixKeyAlreadyExists
	} else if !errors.Is(err, ErrPixKeyNotFound) {
		return domain.PixKey{}, err
	}

	if err := s.repository.CreatePixKey(&pixKey); err != nil {
		return domain.PixKey{}, fmt.Errorf("%w: %v", ErrPixKeyPersistenceConflict, err)
	}
	return pixKey, nil
}

func (s Service) LookupPixKey(key string) (domain.LookupResult, error) {
	normalizedKey := domain.NormalizeLookupKey(key)
	if normalizedKey == "" {
		return domain.LookupResult{}, ValidationError{
			Code:    "validation_error",
			Message: "entrada invalida",
			Fields:  map[string]string{"key": "chave Pix e obrigatoria"},
		}
	}
	return s.repository.LookupPixKey(normalizedKey)
}

func (s Service) ListAccountPixKeys(accountIDValue string) ([]domain.PixKey, error) {
	accountID, err := uuid.FromString(accountIDValue)
	if err != nil {
		return nil, ErrInvalidAccountIdentifier
	}
	return s.repository.ListPixKeysByAccount(accountID)
}
