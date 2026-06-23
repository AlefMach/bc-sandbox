package postgres

import (
	"errors"

	accountdomain "bc_sandbox/internal/accounts/domain"
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

func IsNotFound(err error) bool {
	return errors.Is(err, application.ErrPixKeyNotFound) ||
		errors.Is(err, application.ErrAccountNotFound) ||
		errors.Is(err, application.ErrInvalidAccountIdentifier)
}
