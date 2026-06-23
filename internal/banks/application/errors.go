package application

import "errors"

var (
	ErrBankNotFound              = errors.New("bank not found")
	ErrBankCannotInitiate        = errors.New("banco offline ou inativo nao pode iniciar transacoes")
	ErrBankPersistenceConflict   = errors.New("bank persistence conflict")
	ErrAuditPersistenceConflict  = errors.New("audit persistence conflict")
	ErrInvalidBankIdentifier     = errors.New("invalid bank identifier")
	ErrInvalidBankCreationInput  = errors.New("invalid bank creation input")
	ErrInvalidBankStatusMutation = errors.New("invalid bank status mutation")
)

type ValidationError struct {
	Code    string
	Message string
	Fields  map[string]string
}

func (e ValidationError) Error() string {
	return e.Message
}
