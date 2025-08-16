package math

import (
	"testing"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/stretchr/testify/assert"
)

func TestFixedPoint_MulDown(t *testing.T) {
	t.Parallel()
	t.Run("it should return correct result", func(t *testing.T) {
		result, err := FixedPoint.MulDown(number.NewUint256("25925243203807071591361"), number.NewUint256("176891139771667"))

		assert.Nil(t, err)
		assert.Zero(t, result.Cmp(number.NewUint256("4585945819179096677")))
	})
}

func TestFixedPoint_Complement(t *testing.T) {
	t.Parallel()
	t.Run("it should return correct result", func(t *testing.T) {
		result := FixedPoint.Complement(number.NewUint256("999823108860228333"))

		assert.Zero(t, result.Cmp(number.NewUint256("176891139771667")))
	})
}
