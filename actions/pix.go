package actions

import (
	"errors"
	"net/http"

	pixpostgres "bc_sandbox/internal/pix/adapters/postgres"
	pixapp "bc_sandbox/internal/pix/application"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
)

type pixKeyPayload struct {
	KeyType string `json:"key_type" form:"key_type"`
	Key     string `json:"key" form:"key"`
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

func pixService(c buffalo.Context) pixapp.Service {
	tx := c.Value("tx").(*pop.Connection)
	return pixapp.NewService(pixpostgres.NewRepository(tx))
}

func renderPixError(c buffalo.Context, err error, fallbackCode string, fallbackMessage string) error {
	var validation pixapp.ValidationError
	if errors.As(err, &validation) {
		return renderValidationError(c, validation.Fields)
	}
	if pixpostgres.IsNotFound(err) {
		return renderAPIError(c, http.StatusNotFound, "pix_key_not_found", "chave Pix inexistente", nil)
	}
	if errors.Is(err, pixapp.ErrPixKeyAlreadyExists) {
		return renderAPIError(c, http.StatusConflict, "pix_key_already_exists", "chave Pix ja cadastrada", nil)
	}
	if errors.Is(err, pixapp.ErrAccountCannotReceivePixKey) {
		return renderAPIError(c, http.StatusConflict, "account_cannot_receive_pix_key", "conta bloqueada ou encerrada nao pode receber chave Pix", nil)
	}
	if errors.Is(err, pixapp.ErrPixKeyPersistenceConflict) {
		return renderAPIError(c, http.StatusConflict, fallbackCode, fallbackMessage, err.Error())
	}
	return renderAPIError(c, http.StatusInternalServerError, fallbackCode, fallbackMessage, err.Error())
}
