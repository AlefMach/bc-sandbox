package actions

import (
	"fmt"

	accountdomain "bc_sandbox/internal/accounts/domain"
	"bc_sandbox/internal/banks/domain"
	pixdomain "bc_sandbox/internal/pix/domain"
	"bc_sandbox/public"
	"bc_sandbox/templates"

	"github.com/gobuffalo/buffalo/render"
)

var r *render.Engine

func init() {
	r = render.New(render.Options{
		// HTML layout to be used for all HTML requests:
		HTMLLayout: "application.plush.html",

		// fs.FS containing templates
		TemplatesFS: templates.FS(),

		// fs.FS containing assets
		AssetsFS: public.FS(),

		// Add template helpers here:
		Helpers: render.Helpers{
			"bankStatusLabel":           domain.StatusLabel,
			"accountStatusLabel":        accountdomain.AccountStatusLabel,
			"pixKeyTypeLabel":           pixdomain.KeyTypeLabel,
			"pixKeyStatusLabel":         pixdomain.KeyStatusLabel,
			"pixTransactionStatusLabel": pixdomain.StatusLabel,
			"centsToBRL": func(cents int64) string {
				return fmt.Sprintf("%d,%02d", cents/100, cents%100)
			},
			"sumPending": func(banks []domain.BankWithMetrics) int64 {
				var total int64
				for _, bank := range banks {
					total += bank.Metrics.PendingTransactions
				}
				return total
			},
			"sumCompleted": func(banks []domain.BankWithMetrics) int64 {
				var total int64
				for _, bank := range banks {
					total += bank.Metrics.CompletedTransactions
				}
				return total
			},
			// for non-bootstrap form helpers uncomment the lines
			// below and import "github.com/gobuffalo/helpers/forms"
			// forms.FormKey:     forms.Form,
			// forms.FormForKey:  forms.FormFor,
		},
	})
}
