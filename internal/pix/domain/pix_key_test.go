package domain

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
)

func Test_NewPixKeyNormalizesAndValidatesTypes(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	now := time.Now().UTC()

	cases := []struct {
		name     string
		keyType  string
		value    string
		expected string
	}{
		{name: "cpf", keyType: "CPF", value: "123.456.789-01", expected: "12345678901"},
		{name: "cnpj", keyType: "cnpj", value: "12.345.678/0001-99", expected: "12345678000199"},
		{name: "email", keyType: "e-mail", value: "PIX@Example.com", expected: "pix@example.com"},
		{name: "phone", keyType: "telefone", value: "+55 (11) 99999-0000", expected: "5511999990000"},
		{name: "random", keyType: "aleatoria", value: "client-generated-key", expected: "client-generated-key"},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			pixKey, validation := NewPixKey(accountID, testCase.keyType, testCase.value, now)

			require.Empty(t, validation)
			require.Equal(t, testCase.expected, pixKey.KeyValue)
			require.Equal(t, KeyStatusActive, pixKey.Status)
		})
	}
}

func Test_NewPixKeyRejectsInvalidInput(t *testing.T) {
	_, validation := NewPixKey(uuid.Must(uuid.NewV4()), "email", "invalid-email", time.Now().UTC())

	require.Equal(t, "e-mail invalido", validation["key"])
}
