package missioncontrol

import (
	"errors"
	"strings"
)

type surfacedValidationError struct {
	err      error
	raw      string
	surfaced string
}

func (e surfacedValidationError) Error() string {
	return strings.Replace(e.err.Error(), e.raw, e.surfaced, 1)
}

func (e surfacedValidationError) Unwrap() error {
	return e.err
}

func SurfaceValidationError(err error) error {
	if err == nil {
		return nil
	}

	var validationErr ValidationError
	if !errors.As(err, &validationErr) {
		return err
	}

	canonicalCode := canonicalizeAuditErrorCode(validationErr.Code, validationErr.Message)
	if canonicalCode == "" || canonicalCode == validationErr.Code {
		return err
	}

	raw := validationErr.Error()
	surfaced := validationErr
	surfaced.Code = canonicalCode
	surfacedText := surfaced.Error()
	if raw == surfacedText || !strings.Contains(err.Error(), raw) {
		return err
	}

	return surfacedValidationError{
		err:      err,
		raw:      raw,
		surfaced: surfacedText,
	}
}

func SurfacedValidationErrorString(err error) string {
	if err == nil {
		return ""
	}
	return SurfaceValidationError(err).Error()
}
