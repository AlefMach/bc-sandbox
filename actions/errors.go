package actions

import (
	"net/http"
	"time"

	"github.com/gobuffalo/buffalo"
	"github.com/gofrs/uuid"
)

type apiError struct {
	Code      string      `json:"code"`
	Message   string      `json:"message"`
	Details   interface{} `json:"details"`
	TraceID   string      `json:"trace_id"`
	Timestamp time.Time   `json:"timestamp"`
}

func renderAPIError(c buffalo.Context, status int, code string, message string, details interface{}) error {
	traceID := c.Request().Header.Get("X-Request-Id")
	if traceID == "" {
		traceID = uuid.Must(uuid.NewV4()).String()
	}

	return c.Render(status, r.JSON(apiError{
		Code:      code,
		Message:   message,
		Details:   details,
		TraceID:   traceID,
		Timestamp: time.Now().UTC(),
	}))
}

func renderValidationError(c buffalo.Context, details interface{}) error {
	return renderAPIError(c, http.StatusBadRequest, "validation_error", "entrada invalida", details)
}
