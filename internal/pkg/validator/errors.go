package validator

import "fmt"

type ValidationError struct {
	Field       string
	Description string
}

func NewValidationError(field, description string) *ValidationError {
	return &ValidationError{Field: field, Description: description}
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("[validator.ValidationError] field: [%s] » description: [%s]", e.Field, e.Description)
}
