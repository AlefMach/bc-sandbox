package domain

import "testing"

func TestBankCanInitiateTransactions(t *testing.T) {
	if !(Bank{Status: StatusActive}).CanInitiateTransactions() {
		t.Fatal("active bank should initiate transactions")
	}
	if !(Bank{Status: StatusMaintenance}).CanInitiateTransactions() {
		t.Fatal("maintenance bank should initiate transactions")
	}
	if (Bank{Status: StatusOffline}).CanInitiateTransactions() {
		t.Fatal("offline bank should not initiate transactions")
	}
	if (Bank{Status: StatusInactive}).CanInitiateTransactions() {
		t.Fatal("inactive bank should not initiate transactions")
	}
}

func TestValidateBankInput(t *testing.T) {
	if validation := ValidateBankInput("Banco Teste", "341", StatusActive); len(validation) > 0 {
		t.Fatalf("expected valid bank input, got %#v", validation)
	}
	if validation := ValidateBankInput("", "34", "bloqueado"); len(validation) == 0 {
		t.Fatal("expected validation errors")
	}
	if !ValidStatus("ativo") {
		t.Fatal("expected portuguese active status alias to be valid")
	}
	if got := NormalizeStatus("em manutencao"); got != StatusMaintenance {
		t.Fatalf("expected maintenance status, got %s", got)
	}
}
