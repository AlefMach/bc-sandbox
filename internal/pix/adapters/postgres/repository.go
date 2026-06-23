package postgres

import (
	"errors"

	accountdomain "bc_sandbox/internal/accounts/domain"
	bankapp "bc_sandbox/internal/banks/application"
	bankdomain "bc_sandbox/internal/banks/domain"
	"bc_sandbox/internal/pix/application"
	"bc_sandbox/internal/pix/domain"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type Repository struct {
	tx *pop.Connection
}

func NewRepository(tx *pop.Connection) Repository {
	return Repository{tx: tx}
}

func (r Repository) FindAccountByID(id uuid.UUID) (accountdomain.Account, error) {
	account := accountdomain.Account{}
	if err := r.tx.Find(&account, id); err != nil {
		return accountdomain.Account{}, application.ErrAccountNotFound
	}
	return account, nil
}

func (r Repository) FindBankByID(id uuid.UUID) (bankdomain.Bank, error) {
	bank := bankdomain.Bank{}
	if err := r.tx.Find(&bank, id); err != nil {
		return bankdomain.Bank{}, bankapp.ErrBankNotFound
	}
	return bank, nil
}

func (r Repository) FindPixKey(key string) (domain.PixKey, error) {
	pixKey := domain.PixKey{}
	if err := r.tx.Where("key_value = ?", key).First(&pixKey); err != nil {
		return domain.PixKey{}, application.ErrPixKeyNotFound
	}
	return pixKey, nil
}

func (r Repository) CreatePixKey(pixKey *domain.PixKey) error {
	return r.tx.Create(pixKey)
}

func (r Repository) LookupPixKey(key string) (domain.LookupResult, error) {
	row := struct {
		BankID         uuid.UUID `db:"bank_id"`
		BankName       string    `db:"bank_name"`
		BankCode       string    `db:"bank_code"`
		AccountID      uuid.UUID `db:"account_id"`
		Agency         string    `db:"agency"`
		Number         string    `db:"number"`
		AccountStatus  string    `db:"account_status"`
		HolderID       uuid.UUID `db:"holder_id"`
		HolderName     string    `db:"holder_name"`
		HolderDocument string    `db:"holder_document"`
		KeyType        string    `db:"key_type"`
		Key            string    `db:"key_value"`
		KeyStatus      string    `db:"key_status"`
	}{}
	err := r.tx.RawQuery(`
		SELECT
			banks.id AS bank_id,
			banks.name AS bank_name,
			banks.code AS bank_code,
			accounts.id AS account_id,
			accounts.agency,
			accounts.number,
			accounts.status AS account_status,
			customers.id AS holder_id,
			customers.name AS holder_name,
			customers.document AS holder_document,
			pix_keys.key_type,
			pix_keys.key_value,
			pix_keys.status AS key_status
		FROM pix_keys
		INNER JOIN accounts ON accounts.id = pix_keys.account_id
		INNER JOIN banks ON banks.id = accounts.bank_id
		INNER JOIN customers ON customers.id = accounts.customer_id
		WHERE pix_keys.key_value = ?
	`, key).First(&row)
	if err != nil {
		return domain.LookupResult{}, application.ErrPixKeyNotFound
	}

	return domain.LookupResult{
		BankID:         row.BankID,
		BankName:       row.BankName,
		BankCode:       row.BankCode,
		AccountID:      row.AccountID,
		Agency:         row.Agency,
		Number:         row.Number,
		AccountStatus:  row.AccountStatus,
		HolderID:       row.HolderID,
		HolderName:     row.HolderName,
		HolderDocument: row.HolderDocument,
		KeyType:        row.KeyType,
		Key:            row.Key,
		KeyStatus:      row.KeyStatus,
	}, nil
}

func (r Repository) ListPixKeysByAccount(accountID uuid.UUID) ([]domain.PixKey, error) {
	keys := domain.PixKeys{}
	if err := r.tx.Where("account_id = ?", accountID).Order("created_at desc").All(&keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (r Repository) CreateTransaction(transaction *domain.Transaction) error {
	return r.tx.Create(transaction)
}

func (r Repository) UpdateTransaction(transaction *domain.Transaction) error {
	return r.tx.Update(transaction)
}

func (r Repository) FindTransactionByID(id uuid.UUID) (domain.Transaction, error) {
	transaction := domain.Transaction{}
	if err := r.tx.Find(&transaction, id); err != nil {
		return domain.Transaction{}, application.ErrTransactionNotFound
	}
	return transaction, nil
}

func (r Repository) ListTransactionsByBank(bankID uuid.UUID) ([]domain.Transaction, error) {
	transactions := domain.Transactions{}
	if err := r.tx.Where("payer_bank_id = ? OR receiver_bank_id = ?", bankID, bankID).Order("created_at desc").All(&transactions); err != nil {
		return nil, err
	}
	return transactions, nil
}

func (r Repository) CreateTransactionEvent(event *domain.TransactionEvent) error {
	return r.tx.Create(event)
}

func (r Repository) ListTransactionEvents(transactionID uuid.UUID) ([]domain.TransactionEvent, error) {
	events := domain.TransactionEvents{}
	if err := r.tx.Where("transaction_id = ?", transactionID).Order("created_at asc").All(&events); err != nil {
		return nil, err
	}
	return events, nil
}

func (r Repository) UpdateAccountBalance(account *accountdomain.Account) error {
	return r.tx.Update(account)
}

func IsNotFound(err error) bool {
	return errors.Is(err, application.ErrPixKeyNotFound) ||
		errors.Is(err, application.ErrAccountNotFound) ||
		errors.Is(err, application.ErrTransactionNotFound) ||
		errors.Is(err, application.ErrInvalidTransactionIdentifier) ||
		errors.Is(err, application.ErrInvalidAccountIdentifier) ||
		errors.Is(err, bankapp.ErrBankNotFound)
}
