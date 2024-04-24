package utils

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTenPowDecimalsFloat(t *testing.T) {
	assert.Nil(t, TenPowDecimalsFloat(MIN_TOKEN_DECIMAL-1))
	assert.Nil(t, TenPowDecimalsFloat(MAX_TOKEN_DECIMAL+1))

	for i := MIN_TOKEN_DECIMAL; i <= MAX_TOKEN_DECIMAL; i++ {
		expected := "1" + strings.Repeat("0", i)
		assert.Equal(t, expected, TenPowDecimals[i].String())
		assert.Equal(t, expected, TenPowDecimalsFloat(i).Text('f', 0))
	}
}
