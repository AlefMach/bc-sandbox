package ports

import (
	accountdomain "bc_sandbox/internal/accounts/domain"
	bankdomain "bc_sandbox/internal/banks/domain"
	"bc_sandbox/internal/pix/domain"

	"github.com/gofrs/uuid"
)

type Repository interface {
	FindAccountByID(id uuid.UUID) (accountdomain.Account, error)
	FindBankByID(id uuid.UUID) (bankdomain.Bank, error)
	FindPixKey(key string) (domain.PixKey, error)
	CreatePixKey(pixKey *domain.PixKey) error
	LookupPixKey(key string) (domain.LookupResult, error)
	ListPixKeysByAccount(accountID uuid.UUID) ([]domain.PixKey, error)
	CreateTransaction(transaction *domain.Transaction) error
	UpdateTransaction(transaction *domain.Transaction) error
	FindTransactionByID(id uuid.UUID) (domain.Transaction, error)
	ListTransactionsByBank(bankID uuid.UUID) ([]domain.Transaction, error)
	CreateTransactionEvent(event *domain.TransactionEvent) error
	ListTransactionEvents(transactionID uuid.UUID) ([]domain.TransactionEvent, error)
	UpdateAccountBalance(account *accountdomain.Account) error
}
