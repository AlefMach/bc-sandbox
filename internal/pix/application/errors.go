package application

import "errors"

var (
	ErrPixKeyNotFound                 = errors.New("pix key not found")
	ErrPixKeyAlreadyExists            = errors.New("pix key already exists")
	ErrInvalidAccountIdentifier       = errors.New("invalid account identifier")
	ErrAccountNotFound                = errors.New("account not found")
	ErrAccountCannotReceivePixKey     = errors.New("conta bloqueada ou encerrada nao pode receber chave Pix")
	ErrPixKeyPersistenceConflict      = errors.New("pix key persistence conflict")
	ErrTransactionNotFound            = errors.New("pix transaction not found")
	ErrInvalidTransactionIdentifier   = errors.New("invalid pix transaction identifier")
	ErrPayerAccountCannotOperate      = errors.New("conta pagadora nao pode operar")
	ErrReceiverAccountCannotReceive   = errors.New("conta recebedora nao pode receber Pix")
	ErrOriginBankCannotInitiate       = errors.New("banco origem offline ou inativo nao pode iniciar Pix")
	ErrInsufficientFunds              = errors.New("saldo insuficiente")
	ErrPixSelfTransferNotAllowed      = errors.New("nao e permitido enviar Pix para a propria conta")
	ErrInvalidTransition              = errors.New("transicao de status invalida")
	ErrCentralBankSettlementOnly      = errors.New("somente o Banco Central pode liquidar Pix interbancario")
	ErrTransactionPersistenceConflict = errors.New("pix transaction persistence conflict")
)

type ValidationError struct {
	Code    string
	Message string
	Fields  map[string]string
}

func (e ValidationError) Error() string {
	return e.Message
}
