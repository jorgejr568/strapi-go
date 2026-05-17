package strapi

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Error is the typed error returned for any non-2xx Strapi response. It
// carries the full payload from Strapi's error envelope so callers can
// inspect details. Use errors.Is with the package sentinels for common
// branching.
type Error struct {
	Status  int             `json:"status"`
	Name    string          `json:"name"`
	Message string          `json:"message"`
	Details json.RawMessage `json:"details,omitempty"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("strapi: %s (%d): %s", e.Name, e.Status, e.Message)
}

// Sentinel errors. Pair with errors.Is.
var (
	ErrNotFound     = errors.New("strapi: not found")
	ErrUnauthorized = errors.New("strapi: unauthorized")
	ErrForbidden    = errors.New("strapi: forbidden")
	ErrValidation   = errors.New("strapi: validation error")
	ErrBadResponse  = errors.New("strapi: bad response")
)

func (e *Error) Is(target error) bool {
	switch target {
	case ErrNotFound:
		return e.Status == 404 || e.Name == "NotFoundError"
	case ErrUnauthorized:
		return e.Status == 401 || e.Name == "UnauthorizedError"
	case ErrForbidden:
		return e.Status == 403 || e.Name == "ForbiddenError"
	case ErrValidation:
		return e.Status == 400 || e.Name == "ValidationError"
	}
	return false
}
