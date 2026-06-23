package application

import "errors"

var (
	ErrPixKeyNotFound             = errors.New("pix key not found")
	ErrPixKeyAlreadyExists        = errors.New("pix key already exists")
	ErrInvalidAccountIdentifier   = errors.New("invalid account identifier")
	ErrAccountNotFound            = errors.New("account not found")
	ErrAccountCannotReceivePixKey = errors.New("conta bloqueada ou encerrada nao pode receber chave Pix")
	ErrPixKeyPersistenceConflict  = errors.New("pix key persistence conflict")
)

type ValidationError struct {
	Code    string
	Message string
	Fields  map[string]string
}

func (e ValidationError) Error() string {
	return e.Message
}
