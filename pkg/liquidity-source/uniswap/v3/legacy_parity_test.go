package uniswapv3

// Sequential parity (quote, UpdateState, quote) of native CalcAmtOut[WithPriceLimit] and legacy
// CalcAmountOut[WithPriceLimit], with and without a price limit and the launchpad guards. Remove with the
// legacy methods and the golden data.

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestCalcAmtOutSequentialParity(t *testing.T) {
	t.Parallel()
	poolEntity := new(entity.Pool)
	require.NoError(t, json.Unmarshal([]byte(poolEncoded), poolEntity))
	const uni = "0x1f9840a85d5af5bf1d1762f925bdaddc4201f984"

	plain, err := NewPoolSimulator(*poolEntity, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	var extra ExtraTickU256
	require.NoError(t, json.Unmarshal([]byte(poolEntity.Extra), &extra))
	extra.MaxTxToken, extra.MaxTxAmount = uni, bignumber.NewBig10("50000000000000000000")
	guarded, err := NewPoolSimulatorWithExtra(*poolEntity, &extra, SimulatorConfig{})
	require.NoError(t, err)
	extra.MaxTxToken, extra.MaxTxAmount = "", nil
	extra.BuyRestrictedToken = uni
	restricted, err := NewPoolSimulatorWithExtra(*poolEntity, &extra, SimulatorConfig{})
	require.NoError(t, err)

	amounts := []string{"0", "1", "1000", "100000000000000", "10000000000000000000", "1000000000000000000000",
		"5000000000000000000000000000", new(big.Int).Lsh(big.NewInt(1), 128).String(),
		new(big.Int).Lsh(big.NewInt(1), 255).String(),
		new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1)).String()}
	for name, sim := range map[string]*PoolSimulator{"plain": plain, "max-tx": guarded, "buy-restricted": restricted} {
		for _, limited := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/limit=%v", name, limited), func(t *testing.T) {
				t.Parallel()
				successes := 0
				for dir := range 2 {
					in, out := sim.Info.Tokens[dir], sim.Info.Tokens[1-dir]
					var limit uint256.Int
					if limited { // 0.5% off the current price, on the swap's side
						var delta uint256.Int
						delta.Div(&sim.V3Pool.SqrtRatioX96, uint256.NewInt(200))
						if dir == 0 {
							limit.Sub(&sim.V3Pool.SqrtRatioX96, &delta)
						} else {
							limit.Add(&sim.V3Pool.SqrtRatioX96, &delta)
						}
					}
					for _, amt := range amounts {
						successes += sequentialV3Parity(t, sim, in, out, amt, limit)
					}
				}
				if name == "plain" {
					require.Positive(t, successes, "no quote succeeded; the parity check proves nothing")
				}
			})
		}
	}
}

func sequentialV3Parity(t *testing.T, base *PoolSimulator, tokenIn, tokenOut, amt string, limit uint256.Int) int {
	t.Helper()
	nativeSim := base.CloneState().(*PoolSimulator)
	legacySim := base.CloneState().(*PoolSimulator)
	amtBig, _ := new(big.Int).SetString(amt, 10)
	amtU := uint256.MustFromBig(amtBig)
	successes := 0
	for step := range 3 {
		name := fmt.Sprintf("%s amt=%s step=%d", tokenIn, amt, step)
		lRes, lErr := legacySim.CalcAmountOutWithPriceLimit(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: new(big.Int).Set(amtBig)}, TokenOut: tokenOut}, limit)
		nRes, nErr := nativeSim.CalcAmtOutWithPriceLimit(pool.CalcAmtOutParams{
			TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *amtU}, TokenOut: tokenOut}, limit)
		if lErr != nil || nErr != nil {
			require.Equal(t, fmt.Sprint(lErr), fmt.Sprint(nErr), name+" error")
			require.Nil(t, nRes, name)
			return successes
		}
		successes++
		require.Equal(t, lRes.TokenAmountOut.Amount.String(), nRes.TokenAmountOut.Amount.Dec(), name+" out")
		require.Equal(t, lRes.TokenAmountOut.Token, nRes.TokenAmountOut.Token, name+" out token")
		require.Equal(t, lRes.RemainingTokenAmountIn.Amount.String(), nRes.RemainingTokenAmountIn.Amount.Dec(), name+" remaining")
		require.Equal(t, lRes.RemainingTokenAmountIn.Token, nRes.RemainingTokenAmountIn.Token, name+" remaining token")
		require.Nil(t, lRes.Fee.Amount, name+" legacy leaves fee amount unset")
		require.Equal(t, lRes.Gas, nRes.Gas, name+" gas")
		require.Equal(t, lRes.SwapInfo, nRes.SwapInfo, name+" swapinfo")

		upd := func(sim *PoolSimulator, out *uint256.Int, info any) {
			sim.UpdateState(pool.UpdateStateParams{TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: *amtU},
				TokenAmountOut: pool.TokenAmt{Token: tokenOut, Amount: *out}, SwapInfo: info})
		}
		upd(nativeSim, &nRes.TokenAmountOut.Amount, nRes.SwapInfo)
		upd(legacySim, uint256.MustFromBig(lRes.TokenAmountOut.Amount), lRes.SwapInfo)
	}
	return successes
}
