package uniswapv4

import (
	"errors"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// GrossBeforeFee must exactly invert the exact-in fee deduction: gross leaves at least net after the fee,
// and gross-1 does not.
func TestGrossBeforeFee(t *testing.T) {
	t.Parallel()
	afterFee := func(gross, feeNum, feeDenom *big.Int) *big.Int {
		fee := new(big.Int).Mul(gross, feeNum)
		return fee.Sub(gross, fee.Quo(fee, feeDenom))
	}
	million := big.NewInt(1_000_000)
	for _, fee := range [][2]*big.Int{
		{big.NewInt(0), million},
		{big.NewInt(6000), million},
		{big.NewInt(199_999), million},
		{big.NewInt(9_900), big.NewInt(10_000)},
		{big.NewInt(5_964_214_711_729_622), bignumber.BONE},
	} {
		for _, net := range []int64{1, 2, 3, 99, 994, 1_000_000, 123_456_789_012} {
			gross := GrossBeforeFee(big.NewInt(net), fee[0], fee[1])
			assert.GreaterOrEqual(t, afterFee(gross, fee[0], fee[1]).Int64(), net, "fee %v net %d", fee[0], net)
			assert.Less(t, afterFee(gross.Sub(gross, bignumber.One), fee[0], fee[1]).Int64(), net,
				"fee %v net %d", fee[0], net)
		}
	}
}

// MinAmountIn is how exact-in-only hooks answer CalcIn: the result must reach amountOut and 1 wei less must not,
// wherever the search starts.
func TestMinAmountIn(t *testing.T) {
	t.Parallel()
	errTooSmall, errTooLarge := errors.New("too small"), errors.New("too large")
	minIn, maxIn := big.NewInt(1_000), new(big.Int).Lsh(bignumber.One, 100)
	// out = floor(in * 3 / 7) - 5, erroring outside [minIn, maxIn] like a hook's bound checks
	calcAmountOut := func(amountIn *big.Int) (*big.Int, error) {
		if amountIn.Cmp(minIn) < 0 {
			return nil, errTooSmall
		} else if amountIn.Cmp(maxIn) > 0 {
			return nil, errTooLarge
		}
		out := new(big.Int).Mul(amountIn, big.NewInt(3))
		return out.Sub(out.Quo(out, big.NewInt(7)), big.NewInt(5)), nil
	}

	rng := rand.New(rand.NewPCG(1, 2))
	for range 1000 {
		amountOut := new(big.Int).Lsh(big.NewInt(1+rng.Int64N(1e9)), uint(rng.IntN(60)))
		guess := new(big.Int).Lsh(big.NewInt(1+rng.Int64N(1e9)), uint(rng.IntN(90)))
		amountIn, err := MinAmountIn(amountOut, guess, calcAmountOut)
		require.NoError(t, err, "amountOut %s guess %s", amountOut, guess)

		out, err := calcAmountOut(amountIn)
		require.NoError(t, err)
		require.GreaterOrEqual(t, out.Cmp(amountOut), 0, "amountIn %s gives %s < %s", amountIn, out, amountOut)
		if out, err = calcAmountOut(new(big.Int).Sub(amountIn, bignumber.One)); err == nil {
			require.Negative(t, out.Cmp(amountOut), "amountIn %s is not minimal for %s", amountIn, amountOut)
		}
	}

	// the largest reachable amountOut sits right below the end of the range
	maxOut, _ := calcAmountOut(maxIn)
	amountIn, err := MinAmountIn(maxOut, big.NewInt(1), calcAmountOut)
	require.NoError(t, err)
	out, _ := calcAmountOut(amountIn)
	outLess, _ := calcAmountOut(new(big.Int).Sub(amountIn, bignumber.One))
	assert.True(t, out.Cmp(maxOut) == 0 && outLess.Cmp(maxOut) < 0, "amountIn %s", amountIn)
	_, err = MinAmountIn(new(big.Int).Add(maxOut, bignumber.One), big.NewInt(1), calcAmountOut)
	assert.ErrorIs(t, err, ErrAmountOutUnreachable)
	_, err = MinAmountIn(big.NewInt(0), nil, calcAmountOut)
	assert.ErrorIs(t, err, ErrInvalidAmountOut)
}
