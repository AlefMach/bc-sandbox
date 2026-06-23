package ports

import (
	accountdomain "bc_sandbox/internal/accounts/domain"
	"bc_sandbox/internal/pix/domain"

	"github.com/gofrs/uuid"
)

type Repository interface {
	FindAccountByID(id uuid.UUID) (accountdomain.Account, error)
	FindPixKey(key string) (domain.PixKey, error)
	CreatePixKey(pixKey *domain.PixKey) error
	LookupPixKey(key string) (domain.LookupResult, error)
	ListPixKeysByAccount(accountID uuid.UUID) ([]domain.PixKey, error)
}
