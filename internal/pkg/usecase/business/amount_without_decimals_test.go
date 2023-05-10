package business

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAmountWithoutDecimals(t *testing.T) {
	t.Run("it should return correct amount without decimals", func(t *testing.T) {
		var (
			amount         = big.NewInt(1_234_567_899)
			decimals uint8 = 6
		)

		amountWithoutDecimals := AmountWithoutDecimals(amount, decimals)

		amountWithoutDecimalsFloat64, _ := amountWithoutDecimals.Float64()

		assert.Equal(t, 1234.567899, amountWithoutDecimalsFloat64)
	})
}
