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

func (as *ActionSuite) Test_ProcessPixBetweenBanksCompletesAndCreatesTimeline() {
	originBank := as.createBank("Banco Origem Pix", "121")
	receiverBank := as.createBank("Banco Recebedor Pix", "122")
	originCustomer := as.createCustomer(originBank.ID.String(), "Origem Pix", "12345678901", "origem@example.com")
	receiverCustomer := as.createCustomer(receiverBank.ID.String(), "Recebedor Pix", "12345678902", "recebedor@example.com")
	originAccount := as.createAccount(originBank.ID.String(), originCustomer.ID.String(), "1", "100", 10000, "active")
	receiverAccount := as.createAccount(receiverBank.ID.String(), receiverCustomer.ID.String(), "1", "200", 1500, "active")

	keyResponse := as.JSON("/accounts/%s/pix-keys", receiverAccount.ID).Post(map[string]string{
		"key_type": "email",
		"key":      "recebedor.pix@example.com",
	})
	as.Equal(http.StatusCreated, keyResponse.Code)

	response := as.JSON("/pix-transactions").Post(map[string]interface{}{
		"payer_account_id": originAccount.ID.String(),
		"pix_key":          "recebedor.pix@example.com",
		"amount_cents":     int64(2500),
	})
	as.Equal(http.StatusCreated, response.Code)

	transaction := pixdomain.Transaction{}
	as.NoError(json.Unmarshal(response.Body.Bytes(), &transaction))
	as.Equal(pixdomain.StatusCompleted, transaction.Status)
	as.Equal(originBank.ID, transaction.PayerBankID)
	as.Equal(receiverBank.ID, transaction.ReceiverBankID)

	originBalance := as.JSON("/accounts/%s/balance", originAccount.ID).Get()
	as.Equal(http.StatusOK, originBalance.Code)
	as.Contains(originBalance.Body.String(), `"balance_cents":7500`)
	receiverBalance := as.JSON("/accounts/%s/balance", receiverAccount.ID).Get()
	as.Equal(http.StatusOK, receiverBalance.Code)
	as.Contains(receiverBalance.Body.String(), `"balance_cents":4000`)

	timelineResponse := as.JSON("/pix-transactions/%s/timeline", transaction.ID).Get()
	as.Equal(http.StatusOK, timelineResponse.Code)
	timeline := []pixdomain.TimelineItem{}
	as.NoError(json.Unmarshal(timelineResponse.Body.Bytes(), &timeline))
	as.Len(timeline, 11)
	as.Equal(pixdomain.StatusCreated, timeline[0].Status)
	as.Equal(pixdomain.StatusCompleted, timeline[10].Status)
}

func (as *ActionSuite) Test_ProcessPixFailsWhenInsufficientFunds() {
	originBank := as.createBank("Banco Origem Sem Saldo", "123")
	receiverBank := as.createBank("Banco Recebedor Sem Saldo", "124")
	originCustomer := as.createCustomer(originBank.ID.String(), "Origem Sem Saldo", "12345678903", "origem-sem-saldo@example.com")
	receiverCustomer := as.createCustomer(receiverBank.ID.String(), "Recebedor Sem Saldo", "12345678904", "recebedor-sem-saldo@example.com")
	originAccount := as.createAccount(originBank.ID.String(), originCustomer.ID.String(), "1", "300", 100, "active")
	receiverAccount := as.createAccount(receiverBank.ID.String(), receiverCustomer.ID.String(), "1", "400", 0, "active")

	keyResponse := as.JSON("/accounts/%s/pix-keys", receiverAccount.ID).Post(map[string]string{
		"key_type": "email",
		"key":      "sem.saldo@example.com",
	})
	as.Equal(http.StatusCreated, keyResponse.Code)

	response := as.JSON("/pix-transactions").Post(map[string]interface{}{
		"payer_account_id": originAccount.ID.String(),
		"pix_key":          "sem.saldo@example.com",
		"amount_cents":     int64(2500),
	})
	as.Equal(http.StatusConflict, response.Code)
	as.Contains(response.Body.String(), `"code":"insufficient_funds"`)

	originBalance := as.JSON("/accounts/%s/balance", originAccount.ID).Get()
	as.Contains(originBalance.Body.String(), `"balance_cents":100`)
	receiverBalance := as.JSON("/accounts/%s/balance", receiverAccount.ID).Get()
	as.Contains(receiverBalance.Body.String(), `"balance_cents":0`)
}

func (as *ActionSuite) Test_ProcessPixRejectsSamePayerAndReceiverAccount() {
	bank := as.createBank("Banco Auto Pix", "129")
	customer := as.createCustomer(bank.ID.String(), "Auto Pix", "12345678909", "auto-pix@example.com")
	account := as.createAccount(bank.ID.String(), customer.ID.String(), "1", "900", 10000, "active")

	keyResponse := as.JSON("/accounts/%s/pix-keys", account.ID).Post(map[string]string{
		"key_type": "email",
		"key":      "auto.pix@example.com",
	})
	as.Equal(http.StatusCreated, keyResponse.Code)

	response := as.JSON("/pix-transactions").Post(map[string]interface{}{
		"payer_account_id": account.ID.String(),
		"pix_key":          "auto.pix@example.com",
		"amount_cents":     int64(2500),
	})
	as.Equal(http.StatusConflict, response.Code)
	as.Contains(response.Body.String(), `"code":"pix_self_transfer_not_allowed"`)

	balance := as.JSON("/accounts/%s/balance", account.ID).Get()
	as.Equal(http.StatusOK, balance.Code)
	as.Contains(balance.Body.String(), `"balance_cents":10000`)
}

func (as *ActionSuite) Test_ProcessPixRejectsInactiveOriginBank() {
	originBank := as.createBank("Banco Origem Inativo", "125")
	statusResponse := as.JSON("/banks/%s/status", originBank.ID).Patch(map[string]string{"status": "inactive"})
	as.Equal(http.StatusOK, statusResponse.Code)
	receiverBank := as.createBank("Banco Recebedor Ativo", "126")
	originCustomer := as.createCustomer(originBank.ID.String(), "Origem Inativa", "12345678905", "origem-inativa@example.com")
	receiverCustomer := as.createCustomer(receiverBank.ID.String(), "Recebedor Ativo", "12345678906", "recebedor-ativo@example.com")
	originAccount := as.createAccount(originBank.ID.String(), originCustomer.ID.String(), "1", "500", 10000, "active")
	receiverAccount := as.createAccount(receiverBank.ID.String(), receiverCustomer.ID.String(), "1", "600", 0, "active")

	keyResponse := as.JSON("/accounts/%s/pix-keys", receiverAccount.ID).Post(map[string]string{
		"key_type": "email",
		"key":      "banco.inativo@example.com",
	})
	as.Equal(http.StatusCreated, keyResponse.Code)

	response := as.JSON("/pix-transactions").Post(map[string]interface{}{
		"payer_account_id": originAccount.ID.String(),
		"pix_key":          "banco.inativo@example.com",
		"amount_cents":     int64(500),
	})
	as.Equal(http.StatusConflict, response.Code)
	as.Contains(response.Body.String(), `"code":"origin_bank_cannot_initiate"`)
}

func (as *ActionSuite) Test_InvalidTransitionIsRejectedAndAudited() {
	originBank := as.createBank("Banco Origem Transicao", "127")
	receiverBank := as.createBank("Banco Recebedor Transicao", "128")
	originCustomer := as.createCustomer(originBank.ID.String(), "Origem Transicao", "12345678907", "origem-transicao@example.com")
	receiverCustomer := as.createCustomer(receiverBank.ID.String(), "Recebedor Transicao", "12345678908", "recebedor-transicao@example.com")
	originAccount := as.createAccount(originBank.ID.String(), originCustomer.ID.String(), "1", "700", 10000, "active")
	receiverAccount := as.createAccount(receiverBank.ID.String(), receiverCustomer.ID.String(), "1", "800", 0, "active")

	keyResponse := as.JSON("/accounts/%s/pix-keys", receiverAccount.ID).Post(map[string]string{
		"key_type": "email",
		"key":      "transicao@example.com",
	})
	as.Equal(http.StatusCreated, keyResponse.Code)
	response := as.JSON("/pix-transactions").Post(map[string]interface{}{
		"payer_account_id": originAccount.ID.String(),
		"pix_key":          "transicao@example.com",
		"amount_cents":     int64(500),
	})
	as.Equal(http.StatusCreated, response.Code)
	transaction := pixdomain.Transaction{}
	as.NoError(json.Unmarshal(response.Body.Bytes(), &transaction))

	invalid := as.JSON("/pix-transactions/%s/status", transaction.ID).Patch(map[string]string{
		"status":  pixdomain.StatusSettled,
		"service": pixdomain.ServiceOriginBank,
	})
	as.Equal(http.StatusForbidden, invalid.Code)
	as.Contains(invalid.Body.String(), `"code":"central_bank_settlement_only"`)

	timeline := as.JSON("/pix-transactions/%s/timeline", transaction.ID).Get()
	as.Equal(http.StatusOK, timeline.Code)
	as.Contains(timeline.Body.String(), "somente o Banco Central")
	as.Contains(timeline.Body.String(), "attempted_status")
}
