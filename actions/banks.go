package actions

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	bankpostgres "bc_sandbox/internal/banks/adapters/postgres"
	"bc_sandbox/internal/banks/application"
	pixdomain "bc_sandbox/internal/pix/domain"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type bankPayload struct {
	Name   string `json:"name" form:"name"`
	Code   string `json:"code" form:"code"`
	Status string `json:"status" form:"status"`
}

type statusPayload struct {
	Status string `json:"status" form:"status"`
}

func CreateBank(c buffalo.Context) error {
	payload := bankPayload{}
	if err := bindPayload(c, &payload); err != nil {
		return renderAPIError(c, http.StatusBadRequest, "invalid_json", "corpo da requisicao invalido", err.Error())
	}

	bank, err := bankService(c).CreateBank(application.CreateBankCommand{
		Name:   payload.Name,
		Code:   payload.Code,
		Status: payload.Status,
	})
	if err != nil {
		return renderBankError(c, err, "bank_create_failed", "nao foi possivel cadastrar banco")
	}

	return c.Render(http.StatusCreated, r.JSON(bank))
}

func ListBanks(c buffalo.Context) error {
	banks, err := bankService(c).ListBanks()
	if err != nil {
		return err
	}
	return c.Render(http.StatusOK, r.JSON(banks))
}

func ShowBank(c buffalo.Context) error {
	bank, err := bankService(c).GetBank(c.Param("id"))
	if err != nil {
		return renderBankError(c, err, "bank_not_found", "banco nao encontrado")
	}
	return c.Render(http.StatusOK, r.JSON(bank))
}

func UpdateBankStatus(c buffalo.Context) error {
	payload := statusPayload{}
	if err := bindPayload(c, &payload); err != nil {
		return renderAPIError(c, http.StatusBadRequest, "invalid_json", "corpo da requisicao invalido", err.Error())
	}

	bank, err := bankService(c).ChangeBankStatus(application.ChangeStatusCommand{
		BankID: c.Param("id"),
		Status: payload.Status,
	})
	if err != nil {
		return renderBankError(c, err, "bank_status_update_failed", "nao foi possivel alterar status do banco")
	}

	return c.Render(http.StatusOK, r.JSON(bank))
}

func CentralBankDashboard(c buffalo.Context) error {
	banks, err := bankService(c).ListBanks()
	if err != nil {
		return err
	}
	c.Set("banks", banks)
	return c.Render(http.StatusOK, r.HTML("banks/central.plush.html"))
}

func AdminBanksDashboard(c buffalo.Context) error {
	banks, err := bankService(c).ListBanks()
	if err != nil {
		return err
	}
	c.Set("banks", banks)
	return c.Render(http.StatusOK, r.HTML("banks/admin.plush.html"))
}

func BankDashboard(c buffalo.Context) error {
	bank, err := bankService(c).GetBank(c.Param("id"))
	if err != nil {
		return renderBankError(c, err, "bank_not_found", "banco nao encontrado")
	}
	customers, err := accountService(c).ListCustomers(c.Param("id"))
	if err != nil {
		return renderAccountError(c, err, "customer_list_failed", "nao foi possivel listar clientes")
	}
	accounts, err := accountService(c).ListAccounts(c.Param("id"))
	if err != nil {
		return renderAccountError(c, err, "account_list_failed", "nao foi possivel listar contas")
	}
	pixKeys := pixdomain.PixKeys{}
	for _, account := range accounts {
		accountPixKeys, err := pixService(c).ListAccountPixKeys(account.ID.String())
		if err != nil {
			return renderPixError(c, err, "pix_key_list_failed", "nao foi possivel listar chaves Pix")
		}
		for _, pixKey := range accountPixKeys {
			pixKeys = append(pixKeys, pixKey)
		}
	}
	pixTransactions, err := pixService(c).ListBankTransactions(c.Param("id"))
	if err != nil {
		return renderPixError(c, err, "pix_transaction_list_failed", "nao foi possivel listar transacoes Pix")
	}
	c.Set("bank", bank)
	c.Set("customers", customers)
	c.Set("accounts", accounts)
	c.Set("pixKeys", pixKeys)
	c.Set("pixTransactions", pixTransactions)
	return c.Render(http.StatusOK, r.HTML("banks/show.plush.html"))
}

func bindPayload(c buffalo.Context, destination interface{}) error {
	contentType := c.Request().Header.Get("Content-Type")
	if strings.Contains(contentType, "application/json") {
		return json.NewDecoder(c.Request().Body).Decode(destination)
	}
	return c.Bind(destination)
}

func bankService(c buffalo.Context) application.Service {
	tx := c.Value("tx").(*pop.Connection)
	repository := bankpostgres.NewRepository(tx)
	return application.NewService(repository)
}

func EnsureBankCanInitiateTransactions(tx *pop.Connection, bankID uuid.UUID) error {
	service := application.NewService(bankpostgres.NewRepository(tx))
	return service.EnsureBankCanInitiateTransactions(bankID)
}

func renderBankError(c buffalo.Context, err error, fallbackCode string, fallbackMessage string) error {
	var validation application.ValidationError
	if errors.As(err, &validation) {
		return renderValidationError(c, validation.Fields)
	}
	if bankpostgres.IsNotFound(err) {
		return renderAPIError(c, http.StatusNotFound, "bank_not_found", "banco nao encontrado", nil)
	}
	if errors.Is(err, application.ErrBankPersistenceConflict) || errors.Is(err, application.ErrAuditPersistenceConflict) {
		return renderAPIError(c, http.StatusConflict, fallbackCode, fallbackMessage, err.Error())
	}
	return renderAPIError(c, http.StatusInternalServerError, fallbackCode, fallbackMessage, err.Error())
}
