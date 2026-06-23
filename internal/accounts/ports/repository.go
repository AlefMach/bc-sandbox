package ports

import (
	accounts "bc_sandbox/internal/accounts/domain"
	banks "bc_sandbox/internal/banks/domain"

	"github.com/gofrs/uuid"
)

type Repository interface {
	FindBankByID(id uuid.UUID) (banks.Bank, error)
	CreateCustomer(customer *accounts.Customer) error
	ListCustomersByBank(bankID uuid.UUID) ([]accounts.Customer, error)
	FindCustomerByBank(bankID uuid.UUID, customerID uuid.UUID) (accounts.Customer, error)
	CreateAccount(account *accounts.Account) error
	ListAccountsByBank(bankID uuid.UUID) ([]accounts.Account, error)
	FindAccountByID(id uuid.UUID) (accounts.Account, error)
	FindBalance(accountID uuid.UUID) (accounts.Balance, error)
}
