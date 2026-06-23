package actions

import (
	"encoding/json"
	"net/http"

	"bc_sandbox/internal/banks/domain"
)

func (as *ActionSuite) Test_CreateAndListBanks() {
	createResponse := as.JSON("/banks").Post(map[string]string{
		"name": "Banco Teste",
		"code": "341",
	})

	as.Equal(http.StatusCreated, createResponse.Code)

	created := domain.Bank{}
	as.NoError(json.Unmarshal(createResponse.Body.Bytes(), &created))
	as.Equal("Banco Teste", created.Name)
	as.Equal("341", created.Code)
	as.Equal(domain.StatusActive, created.Status)

	auditCount, err := as.DB.Count(&domain.AuditEvent{})
	as.NoError(err)
	as.Equal(1, auditCount)

	listResponse := as.JSON("/banks").Get()
	as.Equal(http.StatusOK, listResponse.Code)
	as.Contains(listResponse.Body.String(), `"metrics"`)
	as.Contains(listResponse.Body.String(), `"accounts":0`)
}

func (as *ActionSuite) Test_UpdateBankStatusCreatesAuditEvent() {
	createResponse := as.JSON("/banks").Post(map[string]string{
		"name": "Banco Status",
		"code": "033",
	})
	as.Equal(http.StatusCreated, createResponse.Code)

	bank := domain.Bank{}
	as.NoError(json.Unmarshal(createResponse.Body.Bytes(), &bank))

	statusResponse := as.JSON("/banks/%s/status", bank.ID).Patch(map[string]string{
		"status": "offline",
	})
	as.Equal(http.StatusOK, statusResponse.Code)

	updated := domain.Bank{}
	as.NoError(json.Unmarshal(statusResponse.Body.Bytes(), &updated))
	as.Equal(domain.StatusOffline, updated.Status)
	as.False(updated.CanInitiateTransactions())

	auditCount, err := as.DB.Count(&domain.AuditEvent{})
	as.NoError(err)
	as.Equal(2, auditCount)
}

func (as *ActionSuite) Test_CreateBankValidationError() {
	response := as.JSON("/banks").Post(map[string]string{
		"name": "",
		"code": "99",
	})

	as.Equal(http.StatusBadRequest, response.Code)
	as.Contains(response.Body.String(), `"code":"validation_error"`)
	as.Contains(response.Body.String(), `"trace_id"`)
}
