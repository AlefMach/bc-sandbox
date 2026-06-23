package grifts

import (
	"bc_sandbox/models"

	"github.com/gobuffalo/grift/grift"
)

var _ = grift.Namespace("db", func() {

	grift.Desc("seed", "Seeds a database")
	grift.Add("seed", func(c *grift.Context) error {
		return models.DB.RawQuery(`
			INSERT INTO banks (id, name, code, status, created_at, updated_at)
			VALUES
				('00000000-0000-4000-8000-000000000001', 'Banco Central Sandbox', '001', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
				('00000000-0000-4000-8000-000000000237', 'Banco Participante 237', '237', 'active', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
			ON CONFLICT (code) DO NOTHING
		`).Exec()
	})

})
