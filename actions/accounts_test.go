package actions

import (
	"encoding/json"
	"net/http"

	accountdomain "bc_sandbox/internal/accounts/domain"
	bankdomain "bc_sandbox/internal/banks/domain"
)

func (as *ActionSuite) Test_CreateCustomerAndListCustomers() {
	bank := as.createBank("Banco Clientes", "104")

	response := as.JSON("/banks/%s/customers", bank.ID).Post(map[string]string{
		"name":     "Maria Silva",
		"document": "123.456.789-01",
		"email":    "maria@example.com",
	})
	as.Equal(http.StatusCreated, response.Code)

	customer := accountdomain.Customer{}
	as.NoError(json.Unmarshal(response.Body.Bytes(), &customer))
	as.Equal(bank.ID, customer.BankID)
	as.Equal("12345678901", customer.Document)

	listResponse := as.JSON("/banks/%s/customers", bank.ID).Get()
	as.Equal(http.StatusOK, listResponse.Code)
	as.Contains(listResponse.Body.String(), "Maria Silva")
}

func (as *ActionSuite) Test_CreateAccountAndGetBalance() {
	bank := as.createBank("Banco Contas", "756")
	customer := as.createCustomer(bank.ID.String(), "Joao Conta", "12345678901", "joao@example.com")

	response := as.JSON("/banks/%s/accounts", bank.ID).Post(map[string]interface{}{
		"customer_id":   customer.ID.String(),
		"agency":        "0001",
		"number":        "12345",
		"balance_cents": int64(12345),
		"status":        "ativa",
	})
	as.Equal(http.StatusCreated, response.Code)

	account := accountdomain.Account{}
	as.NoError(json.Unmarshal(response.Body.Bytes(), &account))
	as.Equal(bank.ID, account.BankID)
	as.Equal(customer.ID, account.CustomerID)
	as.Equal(accountdomain.AccountStatusActive, account.Status)

	balanceResponse := as.JSON("/accounts/%s/balance", account.ID).Get()
	as.Equal(http.StatusOK, balanceResponse.Code)
	as.Contains(balanceResponse.Body.String(), `"balance_cents":12345`)
	as.Contains(balanceResponse.Body.String(), `"customer_name":"Joao Conta"`)
	as.Contains(balanceResponse.Body.String(), `"status":"active"`)
}

func (as *ActionSuite) Test_CreateAccountRejectsInvalidInput() {
	bank := as.createBank("Banco Validacao", "422")
	customer := as.createCustomer(bank.ID.String(), "Ana Cliente", "12345678901", "ana@example.com")

	response := as.JSON("/banks/%s/accounts", bank.ID).Post(map[string]interface{}{
		"customer_id":   customer.ID.String(),
		"agency":        "",
		"number":        "A12",
		"balance_cents": int64(-1),
		"status":        "suspensa",
	})

	as.Equal(http.StatusBadRequest, response.Code)
	as.Contains(response.Body.String(), `"code":"validation_error"`)
	as.Contains(response.Body.String(), "saldo nao pode ser negativo")
}

func (as *ActionSuite) Test_EnsureAccountCanOperateRejectsBlockedAndClosed() {
	bank := as.createBank("Banco Bloqueio", "389")
	customer := as.createCustomer(bank.ID.String(), "Bia Bloqueio", "12345678901", "bia@example.com")
	blocked := as.createAccount(bank.ID.String(), customer.ID.String(), "1", "10", 0, "blocked")
	closed := as.createAccount(bank.ID.String(), customer.ID.String(), "1", "11", 0, "closed")
	active := as.createAccount(bank.ID.String(), customer.ID.String(), "1", "12", 0, "active")

	as.Error(EnsureAccountCanOperate(as.DB, blocked.ID))
	as.Error(EnsureAccountCanOperate(as.DB, closed.ID))
	as.NoError(EnsureAccountCanOperate(as.DB, active.ID))
}

func (as *ActionSuite) createBank(name string, code string) bankdomain.Bank {
	response := as.JSON("/banks").Post(map[string]string{"name": name, "code": code})
	as.Equal(http.StatusCreated, response.Code)
	bank := bankdomain.Bank{}
	as.NoError(json.Unmarshal(response.Body.Bytes(), &bank))
	return bank
}

func (as *ActionSuite) createCustomer(bankID string, name string, document string, email string) accountdomain.Customer {
	response := as.JSON("/banks/%s/customers", bankID).Post(map[string]string{
		"name":     name,
		"document": document,
		"email":    email,
	})
	as.Equal(http.StatusCreated, response.Code)
	customer := accountdomain.Customer{}
	as.NoError(json.Unmarshal(response.Body.Bytes(), &customer))
	return customer
}

func (as *ActionSuite) createAccount(bankID string, customerID string, agency string, number string, balanceCents int64, status string) accountdomain.Account {
	response := as.JSON("/banks/%s/accounts", bankID).Post(map[string]interface{}{
		"customer_id":   customerID,
		"agency":        agency,
		"number":        number,
		"balance_cents": balanceCents,
		"status":        status,
	})
	as.Equal(http.StatusCreated, response.Code)
	account := accountdomain.Account{}
	as.NoError(json.Unmarshal(response.Body.Bytes(), &account))
	return account
}
