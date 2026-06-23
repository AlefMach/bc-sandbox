package postgres

import (
	"errors"

	"bc_sandbox/internal/banks/application"
	"bc_sandbox/internal/banks/domain"

	"github.com/gobuffalo/pop/v6"
	"github.com/gofrs/uuid"
)

type Repository struct {
	tx *pop.Connection
}

func NewRepository(tx *pop.Connection) Repository {
	return Repository{tx: tx}
}

func (r Repository) Create(bank *domain.Bank) error {
	return r.tx.Create(bank)
}

func (r Repository) FindByID(id uuid.UUID) (domain.Bank, error) {
	bank := domain.Bank{}
	if err := r.tx.Find(&bank, id); err != nil {
		return domain.Bank{}, application.ErrBankNotFound
	}
	return bank, nil
}

func (r Repository) ListWithMetrics() ([]domain.BankWithMetrics, error) {
	banks := domain.Banks{}
	if err := r.tx.Order("name asc").All(&banks); err != nil {
		return nil, err
	}

	result := make([]domain.BankWithMetrics, 0, len(banks))
	for _, bank := range banks {
		metrics, err := r.metricsForBank(bank.ID)
		if err != nil {
			return nil, err
		}
		result = append(result, domain.BankWithMetrics{Bank: bank, Metrics: metrics})
	}
	return result, nil
}

func (r Repository) FindWithMetrics(id uuid.UUID) (domain.BankWithMetrics, error) {
	bank, err := r.FindByID(id)
	if err != nil {
		return domain.BankWithMetrics{}, err
	}
	metrics, err := r.metricsForBank(bank.ID)
	if err != nil {
		return domain.BankWithMetrics{}, err
	}
	return domain.BankWithMetrics{Bank: bank, Metrics: metrics}, nil
}

func (r Repository) Update(bank *domain.Bank) error {
	return r.tx.Update(bank)
}

func (r Repository) CreateAuditEvent(event *domain.AuditEvent) error {
	return r.tx.Create(event)
}

func (r Repository) metricsForBank(bankID uuid.UUID) (domain.Metrics, error) {
	metrics := domain.Metrics{}

	if r.tableExists("accounts") {
		err := r.tx.RawQuery("SELECT COUNT(*) AS accounts FROM accounts WHERE bank_id = ?", bankID).First(&metrics)
		if err != nil {
			return metrics, err
		}
	}

	if r.tableExists("pix_transactions") {
		err := r.tx.RawQuery(`
			SELECT
				COALESCE(SUM(amount_cents), 0) AS total_transacted_cents,
				COUNT(*) FILTER (WHERE status NOT IN ('completed', 'failed')) AS pending_transactions,
				COUNT(*) FILTER (WHERE status = 'completed') AS completed_transactions,
				COUNT(*) FILTER (WHERE status = 'failed') AS failed_transactions
			FROM pix_transactions
			WHERE payer_bank_id = ? OR receiver_bank_id = ?
		`, bankID, bankID).First(&metrics)
		if err != nil {
			return metrics, err
		}
	}

	return metrics, nil
}

func (r Repository) tableExists(tableName string) bool {
	row := struct {
		Exists bool `db:"exists"`
	}{}
	err := r.tx.RawQuery("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = ?) AS exists", tableName).First(&row)
	return err == nil && row.Exists
}

func IsNotFound(err error) bool {
	return errors.Is(err, application.ErrBankNotFound) || errors.Is(err, application.ErrInvalidBankIdentifier)
}
