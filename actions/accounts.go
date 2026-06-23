package actions

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	accountpostgres "bc_sandbox/internal/accounts/adapters/postgres"
	accountapp "bc_sandbox/internal/accounts/application"

	"github.com/gobuffalo/buffalo"
	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type customerPayload struct {
	Name     string `json:"name" form:"name"`
	Document string `json:"document" form:"document"`
	Email    string `json:"email" form:"email"`
}

type accountPayload struct {
	CustomerID     string `json:"customer_id" form:"customer_id"`
	Agency         string `json:"agency" form:"agency"`
	Number         string `json:"number" form:"number"`
	BalanceCents   int64  `json:"balance_cents" form:"balance_cents"`
	BalanceReais   string `json:"balance_reais" form:"balance_reais"`
	InitialBalance string `json:"initial_balance" form:"initial_balance"`
	Status         string `json:"status" form:"status"`
}

func CreateCustomer(c buffalo.Context) error {
	payload := customerPayload{}
	if err := bindPayload(c, &payload); err != nil {
		return renderAPIError(c, http.StatusBadRequest, "invalid_json", "corpo da requisicao invalido", err.Error())
	}

	customer, err := accountService(c).CreateCustomer(accountapp.CreateCustomerCommand{
		BankID:   c.Param("bank_id"),
		Name:     payload.Name,
		Document: payload.Document,
		Email:    payload.Email,
	})
	if err != nil {
		return renderAccountError(c, err, "customer_create_failed", "nao foi possivel cadastrar cliente")
	}
	if !isJSONRequest(c) {
		return c.Redirect(http.StatusSeeOther, "/banks/%s/dashboard", c.Param("bank_id"))
	}
	return c.Render(http.StatusCreated, r.JSON(customer))
}

func ListCustomers(c buffalo.Context) error {
	customers, err := accountService(c).ListCustomers(c.Param("bank_id"))
	if err != nil {
		return renderAccountError(c, err, "customer_list_failed", "nao foi possivel listar clientes")
	}
	return c.Render(http.StatusOK, r.JSON(customers))
}

func CreateAccount(c buffalo.Context) error {
	payload := accountPayload{}
	if err := bindPayload(c, &payload); err != nil {
		return renderAPIError(c, http.StatusBadRequest, "invalid_json", "corpo da requisicao invalido", err.Error())
	}
	balanceCents, err := resolveBalanceCents(payload)
	if err != nil {
		return renderValidationError(c, map[string]string{"balance_cents": err.Error()})
	}

	account, err := accountService(c).CreateAccount(accountapp.CreateAccountCommand{
		BankID:       c.Param("bank_id"),
		CustomerID:   payload.CustomerID,
		Agency:       payload.Agency,
		Number:       payload.Number,
		BalanceCents: balanceCents,
		Status:       payload.Status,
	})
	if err != nil {
		return renderAccountError(c, err, "account_create_failed", "nao foi possivel criar conta")
	}
	if !isJSONRequest(c) {
		return c.Redirect(http.StatusSeeOther, "/banks/%s/dashboard", c.Param("bank_id"))
	}
	return c.Render(http.StatusCreated, r.JSON(account))
}

func GetAccountBalance(c buffalo.Context) error {
	balance, err := accountService(c).GetBalance(c.Param("id"))
	if err != nil {
		return renderAccountError(c, err, "balance_query_failed", "nao foi possivel consultar saldo")
	}
	return c.Render(http.StatusOK, r.JSON(balance))
}

func EnsureAccountCanOperate(tx *pop.Connection, accountID uuid.UUID) error {
	service := accountapp.NewService(accountpostgres.NewRepository(tx))
	return service.EnsureAccountCanOperate(accountID)
}

func accountService(c buffalo.Context) accountapp.Service {
	tx := c.Value("tx").(*pop.Connection)
	return accountapp.NewService(accountpostgres.NewRepository(tx))
}

func renderAccountError(c buffalo.Context, err error, fallbackCode string, fallbackMessage string) error {
	var validation accountapp.ValidationError
	if errors.As(err, &validation) {
		return renderValidationError(c, validation.Fields)
	}
	if accountpostgres.IsNotFound(err) {
		return renderAPIError(c, http.StatusNotFound, fallbackCode, fallbackMessage, nil)
	}
	if errors.Is(err, accountapp.ErrCustomerPersistenceConflict) || errors.Is(err, accountapp.ErrAccountPersistenceConflict) {
		return renderAPIError(c, http.StatusConflict, fallbackCode, fallbackMessage, err.Error())
	}
	if errors.Is(err, accountapp.ErrAccountCannotOperate) {
		return renderAPIError(c, http.StatusConflict, "account_cannot_operate", "conta bloqueada ou encerrada nao pode operar", nil)
	}
	return renderAPIError(c, http.StatusInternalServerError, fallbackCode, fallbackMessage, err.Error())
}

func resolveBalanceCents(payload accountPayload) (int64, error) {
	if payload.BalanceReais != "" || payload.InitialBalance != "" {
		value := payload.BalanceReais
		if value == "" {
			value = payload.InitialBalance
		}
		return parseBRLMoney(value)
	}
	return payload.BalanceCents, nil
}

func parseBRLMoney(value string) (int64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, nil
	}
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, ",", ".")
	parts := strings.Split(value, ".")
	if len(parts) > 2 {
		return 0, errors.New("saldo invalido")
	}
	reais, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || reais < 0 {
		return 0, errors.New("saldo invalido")
	}
	var cents int64
	if len(parts) == 2 {
		fraction := parts[1]
		if len(fraction) > 2 {
			return 0, errors.New("saldo deve possuir no maximo duas casas decimais")
		}
		for len(fraction) < 2 {
			fraction += "0"
		}
		cents, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, errors.New("saldo invalido")
		}
	}
	return reais*100 + cents, nil
}

func isJSONRequest(c buffalo.Context) bool {
	contentType := c.Request().Header.Get("Content-Type")
	accept := c.Request().Header.Get("Accept")
	return strings.Contains(contentType, "application/json") || strings.Contains(accept, "application/json")
}
