package postgres

import (
	"errors"
	"fmt"

	"bc_sandbox/internal/accounts/application"
	accounts "bc_sandbox/internal/accounts/domain"
	bankapp "bc_sandbox/internal/banks/application"
	banks "bc_sandbox/internal/banks/domain"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type Repository struct {
	tx *pop.Connection
}

func NewRepository(tx *pop.Connection) Repository {
	return Repository{tx: tx}
}

func (r Repository) FindBankByID(id uuid.UUID) (banks.Bank, error) {
	bank := banks.Bank{}
	if err := r.tx.Find(&bank, id); err != nil {
		return banks.Bank{}, application.ErrBankNotFound
	}
	return bank, nil
}

func (r Repository) CreateCustomer(customer *accounts.Customer) error {
	return r.tx.Create(customer)
}

func (r Repository) ListCustomersByBank(bankID uuid.UUID) ([]accounts.Customer, error) {
	customers := accounts.Customers{}
	if err := r.tx.Where("bank_id = ?", bankID).Order("created_at desc").All(&customers); err != nil {
		return nil, err
	}
	return customers, nil
}

func (r Repository) FindCustomerByBank(bankID uuid.UUID, customerID uuid.UUID) (accounts.Customer, error) {
	customer := accounts.Customer{}
	err := r.tx.Where("bank_id = ? AND id = ?", bankID, customerID).First(&customer)
	if err != nil {
		return accounts.Customer{}, application.ErrCustomerNotFound
	}
	return customer, nil
}

func (r Repository) CreateAccount(account *accounts.Account) error {
	return r.tx.Create(account)
}

func (r Repository) ListAccountsByBank(bankID uuid.UUID) ([]accounts.Account, error) {
	accountList := accounts.Accounts{}
	if err := r.tx.Where("bank_id = ?", bankID).Order("created_at desc").All(&accountList); err != nil {
		return nil, err
	}
	return accountList, nil
}

func (r Repository) FindAccountByID(id uuid.UUID) (accounts.Account, error) {
	account := accounts.Account{}
	if err := r.tx.Find(&account, id); err != nil {
		return accounts.Account{}, application.ErrAccountNotFound
	}
	return account, nil
}

func (r Repository) FindBalance(accountID uuid.UUID) (accounts.Balance, error) {
	row := struct {
		AccountID    uuid.UUID `db:"account_id"`
		BankID       uuid.UUID `db:"bank_id"`
		BankName     string    `db:"bank_name"`
		CustomerID   uuid.UUID `db:"customer_id"`
		CustomerName string    `db:"customer_name"`
		Agency       string    `db:"agency"`
		Number       string    `db:"number"`
		BalanceCents int64     `db:"balance_cents"`
		Status       string    `db:"status"`
	}{}
	err := r.tx.RawQuery(`
		SELECT
			accounts.id AS account_id,
			accounts.bank_id,
			banks.name AS bank_name,
			customers.id AS customer_id,
			customers.name AS customer_name,
			accounts.agency,
			accounts.number,
			accounts.balance_cents,
			accounts.status
		FROM accounts
		INNER JOIN banks ON banks.id = accounts.bank_id
		INNER JOIN customers ON customers.id = accounts.customer_id
		WHERE accounts.id = ?
	`, accountID).First(&row)
	if err != nil {
		return accounts.Balance{}, application.ErrAccountNotFound
	}

	return accounts.Balance{
		AccountID:     row.AccountID,
		BankID:        row.BankID,
		BankName:      row.BankName,
		CustomerID:    row.CustomerID,
		CustomerName:  row.CustomerName,
		Agency:        row.Agency,
		Number:        row.Number,
		BalanceCents:  row.BalanceCents,
		BalanceAmount: fmt.Sprintf("%d,%02d", row.BalanceCents/100, row.BalanceCents%100),
		Status:        row.Status,
	}, nil
}

func IsNotFound(err error) bool {
	return errors.Is(err, application.ErrBankNotFound) ||
		errors.Is(err, application.ErrCustomerNotFound) ||
		errors.Is(err, application.ErrAccountNotFound) ||
		errors.Is(err, application.ErrInvalidBankIdentifier) ||
		errors.Is(err, application.ErrInvalidCustomerIdentifier) ||
		errors.Is(err, application.ErrInvalidAccountIdentifier) ||
		errors.Is(err, bankapp.ErrBankNotFound)
}
