package makerpsm

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestGetAmountOut_sellGemNoFee(t *testing.T) {
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L166
	pool100 := newPool(t, big.NewInt(100), big.NewInt(0), big.NewInt(0))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), usdxWAD)}
	out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      DAIAddress,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(100), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)

	// reach dept ceiling
	pool100.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: inAmount, TokenAmountOut: *out.TokenAmountOut, Fee: *out.Fee})
	_, err = testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      DAIAddress,
		})
	})
	require.NotNil(t, err)
	fmt.Println(err)
}

func TestGetAmountOut_sellGemWithFee(t *testing.T) {
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L189
	pool100 := newPool(t, big.NewInt(100), tollOnePct, big.NewInt(0))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), usdxWAD)}
	out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      DAIAddress,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(99), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)
}

func TestGetAmountOut_swapBothNoFee(t *testing.T) {
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L208
	// sell 100 USDX
	pool100 := newPool(t, big.NewInt(100), big.NewInt(0), big.NewInt(0))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), usdxWAD)}
	out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      DAIAddress,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(100), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)

	pool100.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: inAmount, TokenAmountOut: *out.TokenAmountOut, Fee: *out.Fee})

	// then buy back with 40 eth
	inAmount = pool.TokenAmount{Token: DAIAddress, Amount: new(big.Int).Mul(big.NewInt(40), bignumber.BONE)}
	out, err = testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      "USDX",
			Limit:         nil,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(40), usdxWAD), out.TokenAmountOut.Amount)
	assert.Equal(t, "USDX", out.TokenAmountOut.Token)
}

func TestGetAmountOut_swapBothWithFee(t *testing.T) {
	// https://github.com/makerdao/dss-psm/blob/master/src/tests/psm.t.sol#L224
	// sell 100 USDX -> 95 eth
	pool100 := newPool(t, big.NewInt(100), new(big.Int).Mul(big.NewInt(5), tollOnePct), new(big.Int).Mul(big.NewInt(10), tollOnePct))
	inAmount := pool.TokenAmount{Token: "USDX", Amount: new(big.Int).Mul(big.NewInt(100), usdxWAD)}
	out, err := testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      DAIAddress,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(95), bignumber.BONE), out.TokenAmountOut.Amount)
	assert.Equal(t, DAIAddress, out.TokenAmountOut.Token)

	pool100.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: inAmount, TokenAmountOut: *out.TokenAmountOut, Fee: *out.Fee})

	// then buy back with 44 eth -> 40 usdx
	inAmount = pool.TokenAmount{Token: DAIAddress, Amount: new(big.Int).Mul(big.NewInt(44), bignumber.BONE)}
	out, err = testutil.MustConcurrentSafe[*pool.CalcAmountOutResult](t, func() (any, error) {
		return pool100.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: inAmount,
			TokenOut:      "USDX",
			Limit:         nil,
		})
	})
	require.Nil(t, err)
	assert.Equal(t, new(big.Int).Mul(big.NewInt(40), usdxWAD), out.TokenAmountOut.Amount)
	assert.Equal(t, "USDX", out.TokenAmountOut.Token)
}
