package dexv2

import (
	"fmt"
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/swaplimit"
)

// Native CalcAmtOut must match legacy CalcAmountOut on amount out, fee, gas and error, for the same limit.
func TestCalcAmtOutMatchesLegacy(t *testing.T) {
	sim, p := snapshotSimulator(t)
	limit := swaplimit.NewInventory(DexType, swaplimit.ToU256Balances(sim.CalculateLimit()))
	for dir := range 2 {
		for _, a := range []int64{10_000, 123_456, 100_000_000} {
			tokenIn, tokenOut := p.Tokens[dir].Address, p.Tokens[1-dir].Address
			legacy, lerr := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(a)}, TokenOut: tokenOut, Limit: pool.AsSwapLimitBig(limit)})
			var amt uint256.Int
			amt.SetUint64(uint64(a))
			native, nerr := sim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: amt}, TokenOut: tokenOut, Limit: limit})
			require.Equal(t, lerr, nerr, "dir=%d in=%d", dir, a)
			if lerr != nil {
				continue
			}
			require.Equal(t, legacy.TokenAmountOut.Amount.String(), native.TokenAmountOut.Amount.Dec(), "amountOut dir=%d in=%d", dir, a)
			require.Equal(t, legacy.Gas, native.Gas, "gas dir=%d in=%d", dir, a)
		}
	}
}

// A shared budget that cannot cover the quote must fail on both paths with the same error.
func TestCalcAmtOutLimitBindsLikeLegacy(t *testing.T) {
	sim, p := snapshotSimulator(t)
	balances := sim.CalculateLimit()
	for k := range balances {
		balances[k] = big.NewInt(0)
	}
	limit := swaplimit.NewInventory(DexType, swaplimit.ToU256Balances(balances))
	var amt uint256.Int
	amt.SetUint64(100_000_000)
	tokenIn, tokenOut := p.Tokens[0].Address, p.Tokens[1].Address
	_, lerr := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(100_000_000)}, TokenOut: tokenOut, Limit: pool.AsSwapLimitBig(limit)})
	_, nerr := sim.CalcAmtOut(pool.CalcAmtOutParams{
		TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: amt}, TokenOut: tokenOut, Limit: limit})
	require.Error(t, lerr)
	require.Equal(t, lerr, nerr)
}

// Sequential quote -> UpdateBalance chains on two clones, one quoted by legacy and one by native,
// must agree to the wei on out, fee, remaining, gas, errors and the resulting state and limits.
func TestCalcAmtOutSequentialParity(t *testing.T) {
	mks := map[string]func() (*PoolSimulator, entity.Pool){
		"d3-static":   func() (*PoolSimulator, entity.Pool) { return snapshotSimulator(t) },
		"d3-dynamic7": func() (*PoolSimulator, entity.Pool) { return dynamicSnapshot(t, 7) },
		"d4-dynamic":  func() (*PoolSimulator, entity.Pool) { return d4Snapshot(t) },
	}
	steps := []struct {
		dir    int
		amount int64
	}{{0, 100_000_000}, {0, 100_000_000}, {1, 30_000_000}, {0, 7_777_777}, {1, 250_000_000}, {0, 50_000_000_000}, {1, 1}}
	for name, mk := range mks {
		sim, p := mk()
		lSim, nSim := sim.CloneState().(*PoolSimulator), sim.CloneState().(*PoolSimulator)
		lLimit := swaplimit.NewInventory(DexType, swaplimit.ToU256Balances(lSim.CalculateLimit()))
		nLimit := swaplimit.NewInventory(DexType, swaplimit.ToU256Balances(nSim.CalculateLimit()))
		quoted := 0
		for i, s := range steps {
			tokenIn, tokenOut := p.Tokens[s.dir].Address, p.Tokens[1-s.dir].Address
			lIn := pool.TokenAmount{Token: tokenIn, Amount: big.NewInt(s.amount)}
			lRes, lErr := lSim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: lIn, TokenOut: tokenOut, Limit: pool.AsSwapLimitBig(lLimit)})
			nIn := pool.TokenAmt{Token: tokenIn, Amount: *uint256.NewInt(uint64(s.amount))}
			nRes, nErr := nSim.CalcAmtOut(pool.CalcAmtOutParams{TokenAmountIn: nIn, TokenOut: tokenOut, Limit: nLimit})
			msg := fmt.Sprintf("%s step %d", name, i)
			require.Equal(t, lErr, nErr, msg)
			if lErr != nil {
				continue
			}
			quoted++
			require.Equal(t, lRes.TokenAmountOut.Amount.String(), nRes.TokenAmountOut.Amount.Dec(), msg+" out")
			require.Equal(t, lRes.TokenAmountOut.Token, nRes.TokenAmountOut.Token, msg+" out token")
			require.Equal(t, lRes.RemainingTokenAmountIn == nil, nRes.RemainingTokenAmountIn == nil, msg+" remaining nil")
			require.Equal(t, lRes.Gas, nRes.Gas, msg+" gas")
			lSim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: lIn, TokenAmountOut: *lRes.TokenAmountOut, SwapInfo: lRes.SwapInfo, SwapLimitBig: pool.AsSwapLimitBig(lLimit)})
			nSim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: nIn.Amount.ToBig()},
				TokenAmountOut: pool.TokenAmount{Token: tokenOut, Amount: nRes.TokenAmountOut.Amount.ToBig()}, SwapInfo: nRes.SwapInfo, SwapLimitBig: pool.AsSwapLimitBig(nLimit)})
			require.Equal(t, lSim.tokenReserves, nSim.tokenReserves, msg+" reserves")
			require.Equal(t, dumpLimits(lSim), dumpLimits(nSim), msg+" limits")
		}
		require.Positive(t, quoted, name)
	}
}
