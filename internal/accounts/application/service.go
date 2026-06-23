package application

import (
	"fmt"
	"time"

	"bc_sandbox/internal/accounts/domain"
	"bc_sandbox/internal/accounts/ports"

	"github.com/gofrs/uuid"
)

type Service struct {
	repository ports.Repository
	clock      func() time.Time
}

type CreateCustomerCommand struct {
	BankID   string
	Name     string
	Document string
	Email    string
}

type CreateAccountCommand struct {
	BankID       string
	CustomerID   string
	Agency       string
	Number       string
	BalanceCents int64
	Status       string
}

func NewService(repository ports.Repository) Service {
	return Service{
		repository: repository,
		clock: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func (s Service) CreateCustomer(command CreateCustomerCommand) (domain.Customer, error) {
	bankID, err := uuid.FromString(command.BankID)
	if err != nil {
		return domain.Customer{}, ErrInvalidBankIdentifier
	}
	if _, err := s.repository.FindBankByID(bankID); err != nil {
		return domain.Customer{}, err
	}

	customer, validation := domain.NewCustomer(bankID, command.Name, command.Document, command.Email, s.clock())
	if len(validation) > 0 {
		return domain.Customer{}, ValidationError{Code: "validation_error", Message: "entrada invalida", Fields: validation}
	}

	if err := s.repository.CreateCustomer(&customer); err != nil {
		return domain.Customer{}, fmt.Errorf("%w: %v", ErrCustomerPersistenceConflict, err)
	}
	return customer, nil
}

func (s Service) ListCustomers(bankIDValue string) ([]domain.Customer, error) {
	bankID, err := uuid.FromString(bankIDValue)
	if err != nil {
		return nil, ErrInvalidBankIdentifier
	}
	if _, err := s.repository.FindBankByID(bankID); err != nil {
		return nil, err
	}
	return s.repository.ListCustomersByBank(bankID)
}

func (s Service) CreateAccount(command CreateAccountCommand) (domain.Account, error) {
	bankID, err := uuid.FromString(command.BankID)
	if err != nil {
		return domain.Account{}, ErrInvalidBankIdentifier
	}
	customerID, err := uuid.FromString(command.CustomerID)
	if err != nil {
		return domain.Account{}, ErrInvalidCustomerIdentifier
	}
	if _, err := s.repository.FindBankByID(bankID); err != nil {
		return domain.Account{}, err
	}
	if _, err := s.repository.FindCustomerByBank(bankID, customerID); err != nil {
		return domain.Account{}, err
	}

	account, validation := domain.NewAccount(bankID, customerID, command.Agency, command.Number, command.BalanceCents, command.Status, s.clock())
	if len(validation) > 0 {
		return domain.Account{}, ValidationError{Code: "validation_error", Message: "entrada invalida", Fields: validation}
	}

	if err := s.repository.CreateAccount(&account); err != nil {
		return domain.Account{}, fmt.Errorf("%w: %v", ErrAccountPersistenceConflict, err)
	}
	return account, nil
}

func (s Service) ListAccounts(bankIDValue string) ([]domain.Account, error) {
	bankID, err := uuid.FromString(bankIDValue)
	if err != nil {
		return nil, ErrInvalidBankIdentifier
	}
	if _, err := s.repository.FindBankByID(bankID); err != nil {
		return nil, err
	}
	return s.repository.ListAccountsByBank(bankID)
}

func (s Service) GetBalance(accountIDValue string) (domain.Balance, error) {
	accountID, err := uuid.FromString(accountIDValue)
	if err != nil {
		return domain.Balance{}, ErrInvalidAccountIdentifier
	}
	return s.repository.FindBalance(accountID)
}

func (s Service) EnsureAccountCanOperate(accountID uuid.UUID) error {
	account, err := s.repository.FindAccountByID(accountID)
	if err != nil {
		return err
	}
	if !account.CanOperate() {
		return ErrAccountCannotOperate
	}
	return nil
}
