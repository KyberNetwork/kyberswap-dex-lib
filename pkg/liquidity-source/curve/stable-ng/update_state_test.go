package stableng

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestUpdateStateMatchesUpdateBalance checks that UpdateState leaves the same reserves as
// UpdateBalance after the same swap, for several swap sizes.
func TestUpdateStateMatchesUpdateBalance(t *testing.T) {
	t.Parallel()
	var pe entity.Pool
	require.NoError(t, json.Unmarshal([]byte(updateStatePoolJSON), &pe))
	base, err := NewPoolSimulator(pe)
	require.NoError(t, err)

	tokens := base.GetTokens()
	in, out := tokens[0], tokens[1]
	quoted := 0
	for _, amt := range []int64{1_000, 1_000_000, 10_000_000} {
		inAmt := big.NewInt(amt)
		res, err := base.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: in, Amount: inAmt}, TokenOut: out})
		if err != nil {
			continue
		}
		quoted++
		outAmt := res.TokenAmountOut.Amount
		feeAmt := big.NewInt(0)
		if res.Fee != nil && res.Fee.Amount != nil {
			feeAmt = res.Fee.Amount
		}

		big1 := base.CloneState()
		big1.UpdateBalance(pool.UpdateBalanceParams{
			TokenAmountIn:  pool.TokenAmount{Token: in, Amount: inAmt},
			TokenAmountOut: pool.TokenAmount{Token: out, Amount: outAmt},
			Fee:            pool.TokenAmount{Token: out, Amount: feeAmt},
		})

		u := base.CloneState().(*PoolSimulator)
		var inU, outU, feeU uint256.Int
		inU.SetFromBig(inAmt)
		outU.SetFromBig(outAmt)
		feeU.SetFromBig(feeAmt)
		u.UpdateState(pool.UpdateStateParams{
			TokenAmountIn:  pool.TokenAmt{Token: in, Amount: inU},
			TokenAmountOut: pool.TokenAmt{Token: out, Amount: outU},
			Fee:            pool.TokenAmt{Token: out, Amount: feeU},
		})

		require.Equal(t, big1.GetReserves(), u.GetReserves(), "amount %d: reserves differ", amt)
		for i := range u.Reserves {
			var want uint256.Int
			require.False(t, want.SetFromBig(big1.GetReserves()[i]))
			require.Zero(t, u.Reserves[i].Cmp(&want), "amount %d: uint256 reserve %d differs", amt, i)
		}
	}
	require.NotZero(t, quoted, "no swap quoted, so nothing was compared")
}

const updateStatePoolJSON = `{"address":"0xdc40d14accd5629bbfa65d057f175871628d13c7","exchange":"curve-stable-ng","type":"curve-stable-ng","timestamp":1709285278,"reserves":["50980","75958","100000000000000"],"tokens":[{"address":"0xfd086bc7cd5c481dcc9c85ebe478a1c0b69fcbb9","symbol":"USDT","decimals":6,"swappable":true},{"address":"0xff970a61a04b1ca14834a43f5de4533ebddb5cc8","symbol":"USDC.e","decimals":6,"swappable":true}],"extra":"{\"InitialA\":\"20000\",\"FutureA\":\"20000\",\"InitialATime\":0,\"FutureATime\":0,\"SwapFee\":\"4000000\",\"AdminFee\":\"5000000000\",\"OffpegFeeMultiplier\":\"20000000000\",\"RateMultipliers\":[\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"]}","staticExtra":"{\"APrecision\":\"100\"}","blockNumber":185969597}`
