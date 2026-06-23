package actions

import (
	"errors"
	"net/http"

	pixpostgres "bc_sandbox/internal/pix/adapters/postgres"
	pixapp "bc_sandbox/internal/pix/application"
	"bc_sandbox/models"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

type pixKeyPayload struct {
	KeyType string `json:"key_type" form:"key_type"`
	Key     string `json:"key" form:"key"`
}

type pixTransactionPayload struct {
	PayerAccountID string `json:"payer_account_id" form:"payer_account_id"`
	PixKey         string `json:"pix_key" form:"pix_key"`
	AmountCents    int64  `json:"amount_cents" form:"amount_cents"`
	AmountReais    string `json:"amount_reais" form:"amount_reais"`
}

type pixTransitionPayload struct {
	Status  string                 `json:"status" form:"status"`
	Service string                 `json:"service" form:"service"`
	BankID  string                 `json:"bank_id" form:"bank_id"`
	Message string                 `json:"message" form:"message"`
	Meta    map[string]interface{} `json:"metadata"`
}

func CreatePixKey(c buffalo.Context) error {
	payload := pixKeyPayload{}
	if err := bindPayload(c, &payload); err != nil {
		return renderAPIError(c, http.StatusBadRequest, "invalid_json", "corpo da requisicao invalido", err.Error())
	}

	pixKey, err := pixService(c).CreatePixKey(pixapp.CreatePixKeyCommand{
		AccountID: c.Param("account_id"),
		KeyType:   payload.KeyType,
		Key:       payload.Key,
	})
	if err != nil {
		return renderPixError(c, err, "pix_key_create_failed", "nao foi possivel cadastrar chave Pix")
	}
	if !isJSONRequest(c) {
		if referer := c.Request().Referer(); referer != "" {
			return c.Redirect(http.StatusSeeOther, referer)
		}
		return c.Redirect(http.StatusSeeOther, "/central-bank")
	}
	return c.Render(http.StatusCreated, r.JSON(pixKey))
}

func GetPixKey(c buffalo.Context) error {
	result, err := pixService(c).LookupPixKey(c.Param("key"))
	if err != nil {
		return renderPixError(c, err, "pix_key_not_found", "chave Pix inexistente")
	}
	return c.Render(http.StatusOK, r.JSON(result))
}

func ProcessPixTransaction(c buffalo.Context) error {
	payload := pixTransactionPayload{}
	if err := bindPayload(c, &payload); err != nil {
		return renderAPIError(c, http.StatusBadRequest, "invalid_json", "corpo da requisicao invalido", err.Error())
	}
	if payload.AmountCents == 0 && payload.AmountReais != "" {
		amountCents, err := parseBRLMoney(payload.AmountReais)
		if err != nil {
			return renderValidationError(c, map[string]string{"amount_cents": err.Error()})
		}
		payload.AmountCents = amountCents
	}

	transaction, err := pixService(c).ProcessPix(pixapp.ProcessPixCommand{
		PayerAccountID: payload.PayerAccountID,
		PixKey:         payload.PixKey,
		AmountCents:    payload.AmountCents,
	})
	if err != nil {
		if !isJSONRequest(c) && transaction.ID.String() != "" {
			return c.Redirect(http.StatusSeeOther, "/pix-transactions/"+transaction.ID.String())
		}
		return renderPixError(c, err, "pix_transaction_failed", "nao foi possivel processar Pix")
	}
	if !isJSONRequest(c) {
		return c.Redirect(http.StatusSeeOther, "/pix-transactions/"+transaction.ID.String())
	}
	return c.Render(http.StatusCreated, r.JSON(transaction))
}

func GetPixTransaction(c buffalo.Context) error {
	transaction, err := pixService(c).GetTransaction(c.Param("id"))
	if err != nil {
		return renderPixError(c, err, "pix_transaction_not_found", "transacao Pix nao encontrada")
	}
	if isJSONRequest(c) {
		return c.Render(http.StatusOK, r.JSON(transaction))
	}
	timeline, err := pixService(c).Timeline(c.Param("id"))
	if err != nil {
		return renderPixError(c, err, "pix_timeline_failed", "nao foi possivel carregar timeline")
	}
	c.Set("transaction", transaction)
	c.Set("timeline", timeline)
	return c.Render(http.StatusOK, r.HTML("pix/show.plush.html"))
}

func GetPixTransactionTimeline(c buffalo.Context) error {
	timeline, err := pixService(c).Timeline(c.Param("id"))
	if err != nil {
		return renderPixError(c, err, "pix_timeline_failed", "nao foi possivel carregar timeline")
	}
	return c.Render(http.StatusOK, r.JSON(timeline))
}

func TransitionPixTransaction(c buffalo.Context) error {
	payload := pixTransitionPayload{}
	if err := bindPayload(c, &payload); err != nil {
		return renderAPIError(c, http.StatusBadRequest, "invalid_json", "corpo da requisicao invalido", err.Error())
	}
	transaction, err := pixService(c).Transition(pixapp.TransitionCommand{
		TransactionID: c.Param("id"),
		Status:        payload.Status,
		Service:       payload.Service,
		BankID:        payload.BankID,
		Message:       payload.Message,
		Metadata:      payload.Meta,
	})
	if err != nil {
		if errors.Is(err, pixapp.ErrInvalidTransition) || errors.Is(err, pixapp.ErrCentralBankSettlementOnly) {
			_ = auditRejectedPixTransition(c.Param("id"), payload)
		}
		return renderPixError(c, err, "pix_transition_failed", "nao foi possivel avancar transacao Pix")
	}
	return c.Render(http.StatusOK, r.JSON(transaction))
}

func pixService(c buffalo.Context) pixapp.Service {
	tx := c.Value("tx").(*pop.Connection)
	return pixapp.NewService(pixpostgres.NewRepository(tx))
}

func auditRejectedPixTransition(transactionID string, payload pixTransitionPayload) error {
	return models.DB.Transaction(func(tx *pop.Connection) error {
		service := pixapp.NewService(pixpostgres.NewRepository(tx))
		_, _ = service.Transition(pixapp.TransitionCommand{
			TransactionID: transactionID,
			Status:        payload.Status,
			Service:       payload.Service,
			BankID:        payload.BankID,
			Message:       payload.Message,
			Metadata:      payload.Meta,
		})
		return nil
	})
}

func renderPixError(c buffalo.Context, err error, fallbackCode string, fallbackMessage string) error {
	var validation pixapp.ValidationError
	if errors.As(err, &validation) {
		return renderValidationError(c, validation.Fields)
	}
	if pixpostgres.IsNotFound(err) {
		return renderAPIError(c, http.StatusNotFound, fallbackCode, fallbackMessage, nil)
	}
	if errors.Is(err, pixapp.ErrPixKeyAlreadyExists) {
		return renderAPIError(c, http.StatusConflict, "pix_key_already_exists", "chave Pix ja cadastrada", nil)
	}
	if errors.Is(err, pixapp.ErrAccountCannotReceivePixKey) {
		return renderAPIError(c, http.StatusConflict, "account_cannot_receive_pix_key", "conta bloqueada ou encerrada nao pode receber chave Pix", nil)
	}
	if errors.Is(err, pixapp.ErrOriginBankCannotInitiate) {
		return renderAPIError(c, http.StatusConflict, "origin_bank_cannot_initiate", "banco origem offline ou inativo nao pode iniciar Pix", nil)
	}
	if errors.Is(err, pixapp.ErrPayerAccountCannotOperate) {
		return renderAPIError(c, http.StatusConflict, "payer_account_cannot_operate", "conta pagadora bloqueada ou encerrada", nil)
	}
	if errors.Is(err, pixapp.ErrReceiverAccountCannotReceive) {
		return renderAPIError(c, http.StatusConflict, "receiver_account_cannot_receive", "conta recebedora bloqueada ou chave inativa", nil)
	}
	if errors.Is(err, pixapp.ErrInsufficientFunds) {
		return renderAPIError(c, http.StatusConflict, "insufficient_funds", "saldo insuficiente", nil)
	}
	if errors.Is(err, pixapp.ErrPixSelfTransferNotAllowed) {
		return renderAPIError(c, http.StatusConflict, "pix_self_transfer_not_allowed", "nao e permitido enviar Pix para a propria conta", nil)
	}
	if errors.Is(err, pixapp.ErrInvalidTransition) {
		return renderAPIError(c, http.StatusConflict, "invalid_transition", "transicao invalida rejeitada e auditada", nil)
	}
	if errors.Is(err, pixapp.ErrCentralBankSettlementOnly) {
		return renderAPIError(c, http.StatusForbidden, "central_bank_settlement_only", "somente o Banco Central pode liquidar Pix interbancario", nil)
	}
	if errors.Is(err, pixapp.ErrPixKeyPersistenceConflict) || errors.Is(err, pixapp.ErrTransactionPersistenceConflict) {
		return renderAPIError(c, http.StatusConflict, fallbackCode, fallbackMessage, err.Error())
	}
	return renderAPIError(c, http.StatusInternalServerError, fallbackCode, fallbackMessage, err.Error())
}
