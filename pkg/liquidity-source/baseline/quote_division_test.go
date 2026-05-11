package baseline

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeActivePriceReturnsInvalidCurveStateOnZeroPremiumDenominator(t *testing.T) {
	params := CurveParams{
		BLV:          uint256.MustFromDecimal("1000000000000000000"),
		Circ:         uint256.MustFromDecimal("1000000000000000000"),
		Supply:       uint256.NewInt(0),
		Reserves:     uint256.MustFromDecimal("1000000000000000000"),
		TotalSupply:  uint256.MustFromDecimal("1000000000000000000"),
		ConvexityExp: uint256.MustFromDecimal("1000000000000000000"),
	}

	var err error
	require.NotPanics(t, func() {
		_, err = computeActivePrice(params)
	})
	assert.ErrorIs(t, err, errInvalidCurveState)
}

func TestComputeZeroCircSwapReturnsInvalidCurveStateOnZeroBufferReservesDenominator(t *testing.T) {
	params := CurveParams{
		BLV:           uint256.MustFromDecimal("1000000000000000000"),
		Circ:          uint256.NewInt(0),
		Supply:        uint256.NewInt(2),
		SwapFee:       uint256.NewInt(0),
		Reserves:      uint256.MustFromDecimal("3000000000000000000"),
		TotalSupply:   uint256.MustFromDecimal("1000000000000000000"),
		ConvexityExp:  uint256.MustFromDecimal("1000000000000000000"),
		LastInvariant: uint256.MustFromDecimal("1000000000000000000"),
	}

	var err error
	require.NotPanics(t, func() {
		_, _, _, err = computeZeroCircSwap(params, big.NewInt(1))
	})
	assert.ErrorIs(t, err, errInvalidCurveState)
}
