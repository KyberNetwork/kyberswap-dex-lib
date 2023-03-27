package validator

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewValidationError(t *testing.T) {
	t.Run("it should return correct validation error", func(t *testing.T) {
		err := NewValidationError("field", "desc")

		assert.Equal(
			t,
			"[validator.ValidationError] field: [field] » description: [desc]",
			err.Error(),
		)
	})
}
