package actions

import (
	"encoding/json"
	"net/http"

	pixdomain "bc_sandbox/internal/pix/domain"
)

func (as *ActionSuite) Test_CreatePixKeyAndLookup() {
	bank := as.createBank("Banco Pix", "260")
	customer := as.createCustomer(bank.ID.String(), "Paula Pix", "12345678901", "paula@example.com")
	account := as.createAccount(bank.ID.String(), customer.ID.String(), "1", "1000", 0, "active")

	response := as.JSON("/accounts/%s/pix-keys", account.ID).Post(map[string]string{
		"key_type": "email",
		"key":      "Paula.Pix@example.com",
	})
	as.Equal(http.StatusCreated, response.Code)

	pixKey := pixdomain.PixKey{}
	as.NoError(json.Unmarshal(response.Body.Bytes(), &pixKey))
	as.Equal(account.ID, pixKey.AccountID)
	as.Equal(pixdomain.KeyTypeEmail, pixKey.KeyType)
	as.Equal("paula.pix@example.com", pixKey.KeyValue)
	as.Equal(pixdomain.KeyStatusActive, pixKey.Status)

	lookupResponse := as.JSON("/pix-keys/%s", pixKey.KeyValue).Get()
	as.Equal(http.StatusOK, lookupResponse.Code)
	as.Contains(lookupResponse.Body.String(), `"bank_name":"Banco Pix"`)
	as.Contains(lookupResponse.Body.String(), `"bank_code":"260"`)
	as.Contains(lookupResponse.Body.String(), `"holder_name":"Paula Pix"`)
	as.Contains(lookupResponse.Body.String(), `"key_type":"email"`)
	as.Contains(lookupResponse.Body.String(), `"status":"active"`)
}

func (as *ActionSuite) Test_CreatePixKeyRejectsDuplicate() {
	bank := as.createBank("Banco Pix Duplicado", "336")
	customer := as.createCustomer(bank.ID.String(), "Duda Pix", "12345678901", "duda@example.com")
	account := as.createAccount(bank.ID.String(), customer.ID.String(), "1", "2000", 0, "active")

	first := as.JSON("/accounts/%s/pix-keys", account.ID).Post(map[string]string{
		"key_type": "cpf",
		"key":      "123.456.789-01",
	})
	as.Equal(http.StatusCreated, first.Code)

	duplicate := as.JSON("/accounts/%s/pix-keys", account.ID).Post(map[string]string{
		"key_type": "cpf",
		"key":      "12345678901",
	})
	as.Equal(http.StatusConflict, duplicate.Code)
	as.Contains(duplicate.Body.String(), `"code":"pix_key_already_exists"`)
}

func (as *ActionSuite) Test_CreatePixKeyRejectsInactiveAccount() {
	bank := as.createBank("Banco Pix Bloqueio", "341")
	customer := as.createCustomer(bank.ID.String(), "Bruno Pix", "12345678901", "bruno@example.com")
	account := as.createAccount(bank.ID.String(), customer.ID.String(), "1", "3000", 0, "blocked")

	response := as.JSON("/accounts/%s/pix-keys", account.ID).Post(map[string]string{
		"key_type": "phone",
		"key":      "+55 11 99999-0000",
	})
	as.Equal(http.StatusConflict, response.Code)
	as.Contains(response.Body.String(), `"code":"account_cannot_receive_pix_key"`)
}

func (as *ActionSuite) Test_GetPixKeyReturnsStandardErrorWhenMissing() {
	response := as.JSON("/pix-keys/missing-key").Get()

	as.Equal(http.StatusNotFound, response.Code)
	as.Contains(response.Body.String(), `"code":"pix_key_not_found"`)
	as.Contains(response.Body.String(), `"trace_id":`)
	as.Contains(response.Body.String(), `"timestamp":`)
}
