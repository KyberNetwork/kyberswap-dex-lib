package shared

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsDPPMinSwapAmountSupported(t *testing.T) {
	t.Parallel()

	assert.True(t, isDPPMinSwapAmountSupported("DPP 1.1.0"))
	assert.True(t, isDPPMinSwapAmountSupported("DPP Advanced 1.1.0"))

	assert.False(t, isDPPMinSwapAmountSupported("DPP 1.0.0"))
	assert.False(t, isDPPMinSwapAmountSupported("DPP Advanced 1.0.0"))
	assert.False(t, isDPPMinSwapAmountSupported("DPP Oracle 1.1.0"))
	assert.False(t, isDPPMinSwapAmountSupported("DPPOracle Admin 1.1.1"))
}
