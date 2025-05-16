package nomiswapstable

import (
	"fmt"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestCalculateSwap(t *testing.T) {
	t.Parallel()
	// tokens := []string{"0x55d398326f99059fF775485246999027B3197955", "0x8AC76a51cc950d9822D68b83fE1Ad97B32Cd580d"}
	tokenPrecisions := []*uint256.Int{uint256.NewInt(1), uint256.NewInt(1)}
	reserves := []*uint256.Int{uint256.MustFromDecimal("53038106898661241621939"), uint256.MustFromDecimal("75247964820990618778857")}
	amountIn := uint256.MustFromDecimal("1000000000000000000")
	amountOutExpect := uint256.MustFromDecimal("1000031964522011533")
	swapFee := uint256.NewInt(6)
	A := uint256.NewInt(200000)
	out := getAmountOut(amountIn, reserves[0], reserves[1], swapFee, tokenPrecisions[0], tokenPrecisions[1], A)
	fmt.Println(out)
	assert.Equal(t, amountOutExpect, out)
}
