package application

import "errors"

var (
	ErrBankNotFound                = errors.New("bank not found")
	ErrCustomerNotFound            = errors.New("customer not found")
	ErrAccountNotFound             = errors.New("account not found")
	ErrInvalidBankIdentifier       = errors.New("invalid bank identifier")
	ErrInvalidCustomerIdentifier   = errors.New("invalid customer identifier")
	ErrInvalidAccountIdentifier    = errors.New("invalid account identifier")
	ErrCustomerPersistenceConflict = errors.New("customer persistence conflict")
	ErrAccountPersistenceConflict  = errors.New("account persistence conflict")
	ErrAccountCannotOperate        = errors.New("conta bloqueada ou encerrada nao pode operar")
)

type ValidationError struct {
	Code    string
	Message string
	Fields  map[string]string
}

func (e ValidationError) Error() string {
	return e.Message
}
