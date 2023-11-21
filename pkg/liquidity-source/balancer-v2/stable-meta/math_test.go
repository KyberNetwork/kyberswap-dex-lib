package stablemeta

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
)

func Test_calculateInvariant(t *testing.T) {
	t.Run("1. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(5000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("9999991000000000000000"),
			uint256.MustFromDecimal("99999910000000000056"),
			uint256.MustFromDecimal("8897791020011100123456"),
			uint256.MustFromDecimal("13288977911102200123456"),
			uint256.MustFromDecimal("199791011102200123456"),
			uint256.MustFromDecimal("1997200112156340123456"),
		}

		// expected
		expected := "19410511781031881171190"

		// actual
		result, err := StableMath._calculateInvariant(amp, balances, true)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("2. should return error balance didn't converge", func(t *testing.T) {
		// input
		amp := uint256.NewInt(1500)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("1243021482015219478129472423914"),
			uint256.MustFromDecimal("184305801438975139127489247143"),
			uint256.MustFromDecimal("14830215"),
			uint256.MustFromDecimal("3018454729758945"),
			uint256.MustFromDecimal("3145748925789256143057234"),
			uint256.MustFromDecimal("127312951502507043571956954693255219"),
		}

		// actual
		_, err := StableMath._calculateInvariant(amp, balances, true)
		assert.ErrorIs(t, err, ErrStableGetBalanceDidntConverge)
	})

	t.Run("3. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(10000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("214321579579179247"),
			uint256.MustFromDecimal("42179537589172147219"),
			uint256.MustFromDecimal("1520481514459573495"),
			uint256.MustFromDecimal("414759131324123123"),
			uint256.MustFromDecimal("4219759729147925"),
			uint256.MustFromDecimal("5197345436285624443"),
		}

		// expected
		expected := "10463766246264892100"

		// actual
		result, err := StableMath._calculateInvariant(amp, balances, true)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("4. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(5000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("9999991000000310000"),
			uint256.MustFromDecimal("9999991000031400056"),
			uint256.MustFromDecimal("88973215240111123456"),
			uint256.MustFromDecimal("13288977911102513456"),
			uint256.MustFromDecimal("199791414320012356"),
			uint256.MustFromDecimal("1997200112152140156"),
		}

		// expected
		expected := "63504110862071166478"

		// actual
		result, err := StableMath._calculateInvariant(amp, balances, false)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("5. should return error balance didn't converge", func(t *testing.T) {
		// input
		amp := uint256.NewInt(1500)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("1243021482015219478129472423914"),
			uint256.MustFromDecimal("184305801438975139127489247143"),
			uint256.MustFromDecimal("14830215"),
			uint256.MustFromDecimal("3018454729758945"),
			uint256.MustFromDecimal("3145748925789256143057234"),
			uint256.MustFromDecimal("127312951502507043571956954693255219"),
		}

		// actual
		_, err := StableMath._calculateInvariant(amp, balances, false)
		assert.ErrorIs(t, err, ErrStableGetBalanceDidntConverge)
	})

	t.Run("6. should return correct invariant", func(t *testing.T) {
		// input
		amp := uint256.NewInt(10000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("214321579579179247"),
			uint256.MustFromDecimal("42179537589172147219"),
			uint256.MustFromDecimal("1520481514459573495"),
			uint256.MustFromDecimal("414759131324123123"),
			uint256.MustFromDecimal("4219759729147925"),
			uint256.MustFromDecimal("5197345436285624443"),
		}

		// expected
		expected := "10463766246264891996"

		// actual
		result, err := StableMath._calculateInvariant(amp, balances, false)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})
}

func Test_getTokenBalanceGivenInvariantAndAllOtherBalances(t *testing.T) {
	t.Run("1. should return correct balance", func(t *testing.T) {
		// input
		amp := uint256.NewInt(5000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("9999991000000000000000"),
			uint256.MustFromDecimal("99999910000000000056"),
			uint256.MustFromDecimal("8897791020011100123456"),
			uint256.MustFromDecimal("13288977911102200123456"),
			uint256.MustFromDecimal("199791011102200123456"),
			uint256.MustFromDecimal("1997200112156340123456"),
		}
		invariant := uint256.MustFromDecimal("19410511781031881171190")
		tokenIndex := 2

		// expected
		expected := "8897791020011100123930"

		// actual
		result, err := StableMath._getTokenBalanceGivenInvariantAndAllOtherBalances(
			amp,
			balances,
			invariant,
			tokenIndex,
		)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("2. should return correct balance", func(t *testing.T) {
		// input
		amp := uint256.NewInt(25000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("999999100001354743940000000"),
			uint256.MustFromDecimal("999999100018034329147962946"),
			uint256.MustFromDecimal("889779102000123421312964156"),
			uint256.MustFromDecimal("132889779111022001234531231236"),
			uint256.MustFromDecimal("1997910111022512421400123456"),
			uint256.MustFromDecimal("1997200112151432414246340123456"),
		}
		invariant := uint256.MustFromDecimal("194123410511781031881171190")
		tokenIndex := 5

		// expected
		expected := "45669657055"

		// actual
		result, err := StableMath._getTokenBalanceGivenInvariantAndAllOtherBalances(
			amp,
			balances,
			invariant,
			tokenIndex,
		)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("3. should return correct balance", func(t *testing.T) {
		// input
		amp := uint256.NewInt(1000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("99999001354743940000000"),
			uint256.MustFromDecimal("999999100018029147962946"),
			uint256.MustFromDecimal("889779102000123421312964156"),
			uint256.MustFromDecimal("13977922001234531231236"),
			uint256.MustFromDecimal("1997910111022512421400123456"),
			uint256.MustFromDecimal("199720011414246340123456"),
		}
		invariant := uint256.MustFromDecimal("1941234105117810318810")
		tokenIndex := 3

		// expected
		expected := "1"

		// actual
		result, err := StableMath._getTokenBalanceGivenInvariantAndAllOtherBalances(
			amp,
			balances,
			invariant,
			tokenIndex,
		)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})

	t.Run("4. should return error mul overflow", func(t *testing.T) {
		// input
		amp := uint256.NewInt(1000)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("99999001354743"),
			uint256.MustFromDecimal("999999109147962946"),
			uint256.MustFromDecimal("88972000123421312964156"),
			uint256.MustFromDecimal("139701234531231236"),
			uint256.MustFromDecimal("199711022512421400123456"),
			uint256.MustFromDecimal("199720011414246340123456"),
		}
		invariant := uint256.MustFromDecimal("1941234102135117810318810")
		tokenIndex := 2

		// actual
		_, err := StableMath._getTokenBalanceGivenInvariantAndAllOtherBalances(
			amp,
			balances,
			invariant,
			tokenIndex,
		)
		assert.ErrorIs(t, err, math.ErrMulOverflow)
	})

	t.Run("5. should return correct balance", func(t *testing.T) {
		// input
		amp := uint256.NewInt(2222)
		balances := []*uint256.Int{
			uint256.MustFromDecimal("999990011312354743"),
			uint256.MustFromDecimal("999999109147962946"),
			uint256.MustFromDecimal("8897200012342134156"),
			uint256.MustFromDecimal("139701234531231236"),
			uint256.MustFromDecimal("1997110225124214006"),
			uint256.MustFromDecimal("1997200114142123456"),
		}
		invariant := uint256.MustFromDecimal("194123410213511781031")
		tokenIndex := 4

		// expected
		expected := "82106384280816317076136"

		// actual
		result, err := StableMath._getTokenBalanceGivenInvariantAndAllOtherBalances(
			amp,
			balances,
			invariant,
			tokenIndex,
		)
		assert.Nil(t, err)

		// assert
		assert.Equal(t, expected, result.Dec())
	})
}
