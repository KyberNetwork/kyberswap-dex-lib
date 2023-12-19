package gyro2clp

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/math"
)

func TestGyro2CLPMath_calculateInvariant(t *testing.T) {
	t.Run("1. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("13159214729142152142142"),
			uint256.MustFromDecimal("534534538245930253025"),
			uint256.MustFromDecimal("521342019582105821251"),
		}
		sqrtAlpha := uint256.MustFromDecimal("543081905821075215")
		sqrtBeta := uint256.MustFromDecimal("5195812068210482015")
		expected := "8971176444878749653243"
		actual, err := Gyro2CLPMath._calculateInvariant(balances, sqrtAlpha, sqrtBeta)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("2. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("43125104821905712904821042"),
			uint256.MustFromDecimal("13159214729142152142142"),
			uint256.MustFromDecimal("534534538245930253025"),
			uint256.MustFromDecimal("521342019582105821251"),
		}
		sqrtAlpha := uint256.MustFromDecimal("52108329018501701242152")
		sqrtBeta := uint256.MustFromDecimal("5195812068252142410482015")
		expected := "2269942195873188270907700749489"
		actual, err := Gyro2CLPMath._calculateInvariant(balances, sqrtAlpha, sqrtBeta)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("3. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("4219421957210843201572014"),
			uint256.MustFromDecimal("5142304808210743"),
			uint256.MustFromDecimal("19089320158920167015215"),
			uint256.MustFromDecimal("1901283021869012"),
			uint256.MustFromDecimal("5219026821483217049523"),
		}
		sqrtAlpha := uint256.MustFromDecimal("41592723491759215")
		sqrtBeta := uint256.MustFromDecimal("3490189203742501984024823")
		expected := "175497376487372247383893"
		actual, err := Gyro2CLPMath._calculateInvariant(balances, sqrtAlpha, sqrtBeta)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("4. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("10183940218502714"),
			uint256.MustFromDecimal("5129402185201340934"),
			uint256.MustFromDecimal("15910482310521"),
			uint256.MustFromDecimal("59210805342"),
			uint256.MustFromDecimal("53152190482914"),
		}
		sqrtAlpha := uint256.MustFromDecimal("149758975941592723491759215")
		sqrtBeta := uint256.MustFromDecimal("3490189203742501294082304984024823")
		expected := "1525136523614804988163519"
		actual, err := Gyro2CLPMath._calculateInvariant(balances, sqrtAlpha, sqrtBeta)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("5. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("10183940218502714"),
			uint256.MustFromDecimal("5129402185201340934"),
			uint256.MustFromDecimal("15910482310521"),
			uint256.MustFromDecimal("59210805342"),
			uint256.MustFromDecimal("53152190482914"),
		}
		sqrtAlpha := uint256.MustFromDecimal("0")
		sqrtBeta := uint256.MustFromDecimal("3490189203742501294082304984024823")
		expected := "228555300115197454"
		actual, err := Gyro2CLPMath._calculateInvariant(balances, sqrtAlpha, sqrtBeta)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("6. should return correct result", func(t *testing.T) {
		balances := []*uint256.Int{
			// [4321905210421421,51432251214241,31930124850124,1230185209186432553]
			uint256.MustFromDecimal("4321905210421421"),
			uint256.MustFromDecimal("51432251214241"),
			uint256.MustFromDecimal("31930124850124"),
			uint256.MustFromDecimal("1230185209186432553"),
		}
		sqrtAlpha := uint256.MustFromDecimal("5729313213")
		sqrtBeta := uint256.MustFromDecimal("43253214214")
		expected := "1370623894559650057614"
		actual, err := Gyro2CLPMath._calculateInvariant(balances, sqrtAlpha, sqrtBeta)
		assert.Nil(t, err)
		assert.Equal(t, expected, actual.Dec())
	})

	t.Run("7. should return error", func(t *testing.T) {
		balances := []*uint256.Int{
			uint256.MustFromDecimal("10183940218502714"),
			uint256.MustFromDecimal("5129402185201340934"),
			uint256.MustFromDecimal("15910482310521"),
			uint256.MustFromDecimal("59210805342"),
			uint256.MustFromDecimal("53152190482914"),
		}
		sqrtAlpha := uint256.MustFromDecimal("12389173289135721423140")
		sqrtBeta := uint256.MustFromDecimal("0")
		_, err := Gyro2CLPMath._calculateInvariant(balances, sqrtAlpha, sqrtBeta)
		assert.Equal(t, math.ErrZeroDivision, err)
	})
}
