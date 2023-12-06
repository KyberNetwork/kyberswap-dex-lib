package math

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestRPow(t *testing.T) {
	t.Run("1. should return correct value", func(t *testing.T) {
		expected := "2587978906837309703137361322131041"
		x := uint256.MustFromDecimal("3145121")
		n := uint256.MustFromDecimal("10")
		base := uint256.MustFromDecimal("3214")
		res, err := Common.RPow(x, n, base)
		assert.Nil(t, err)
		assert.Equal(t, expected, res.Dec())
	})

	t.Run("2. should return correct value", func(t *testing.T) {
		expected := "0"
		x := uint256.MustFromDecimal("213")
		n := uint256.MustFromDecimal("100")
		base := uint256.MustFromDecimal("999")
		res, err := Common.RPow(x, n, base)
		assert.Nil(t, err)
		assert.Equal(t, expected, res.Dec())
	})

	t.Run("3. should return correct value", func(t *testing.T) {
		expected := "3150676750167233152197066722664292267937844764501205787"
		x := uint256.MustFromDecimal("951")
		n := uint256.MustFromDecimal("59")
		base := uint256.MustFromDecimal("123")
		res, err := Common.RPow(x, n, base)
		assert.Nil(t, err)
		assert.Equal(t, expected, res.Dec())
	})

	t.Run("4. should return correct value", func(t *testing.T) {
		expected := "3150676750167233152197066722664292267937844764501205787"
		x := uint256.MustFromDecimal("951")
		n := uint256.MustFromDecimal("59")
		base := uint256.MustFromDecimal("123")
		res, err := Common.RPow(x, n, base)
		assert.Nil(t, err)
		assert.Equal(t, expected, res.Dec())
	})

	t.Run("5. should return correct value", func(t *testing.T) {
		expected := "123"
		x := uint256.MustFromDecimal("951")
		n := uint256.MustFromDecimal("0")
		base := uint256.MustFromDecimal("123")
		res, err := Common.RPow(x, n, base)
		assert.Nil(t, err)
		assert.Equal(t, expected, res.Dec())
	})

	t.Run("6. should return error", func(t *testing.T) {
		x := uint256.MustFromDecimal("32142194234951")
		n := uint256.MustFromDecimal("3149143124")
		base := uint256.MustFromDecimal("421424")
		_, err := Common.RPow(x, n, base)
		assert.ErrorIs(t, err, ErrOverflow)
	})
}
