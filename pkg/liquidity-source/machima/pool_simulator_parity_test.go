package machima

import (
	"math/big"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// parityCase pins one quote. The golden values come from the big.Int CalcAmountOut before the
// uint256 port, so the native CalcAmtOut must reproduce them exactly, error text included.
type parityCase struct {
	name     string
	sim      func(t *testing.T) *PoolSimulator
	tokenIn  string
	tokenOut string
	amountIn string

	wantOut string
	wantFee string
	wantRem string
	wantGas int64
	wantErr string
}

func taxedSim(t *testing.T) *PoolSimulator {
	return newSim(t, func(e *Extra) { e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300 })
}

func plainSim(t *testing.T) *PoolSimulator {
	return newSim(t, nil)
}

func xmaTaxedSim(t *testing.T) *PoolSimulator {
	return newSimWithTokens(t, wethAddr, xmaAddr, xmaAddr, func(e *Extra) {
		e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300
	})
}

func xmaFloorSim(t *testing.T) *PoolSimulator {
	var floor uint256.Int
	require.NoError(t, uniswapv3.GetSqrtRatioAtTick(500, &floor))
	return newSimWithTokens(t, wethAddr, xmaAddr, xmaAddr, func(e *Extra) {
		e.HasTax, e.BuyTaxBps, e.SellTaxBps = true, 500, 300
		e.XmaSellSqrtPriceLimit = floor.ToBig()
	})
}

func parityCases() []parityCase {
	const oneFinney = "1000000000000000"
	return []parityCase{
		{name: "buy taxed", sim: taxedSim, tokenIn: wethAddr, tokenOut: tokenAddr, amountIn: oneFinney,
			wantOut: "939616290878428", wantFee: "50000000000000", wantRem: "0", wantGas: 352500},
		{name: "sell taxed", sim: taxedSim, tokenIn: tokenAddr, tokenOut: wethAddr, amountIn: oneFinney,
			wantOut: "959350243259174", wantFee: "0", wantRem: "0", wantGas: 350000},
		{name: "buy untaxed pool", sim: plainSim, tokenIn: wethAddr, tokenOut: tokenAddr, amountIn: oneFinney,
			wantOut: "989020869339354", wantFee: "0", wantRem: "0", wantGas: 352500},
		{name: "sell untaxed pool", sim: plainSim, tokenIn: tokenAddr, tokenOut: wethAddr, amountIn: oneFinney,
			wantOut: "989020869339354", wantFee: "0", wantRem: "0", wantGas: 350000},
		{name: "buy tax gate: input taxes to zero", sim: taxedSim, tokenIn: wethAddr, tokenOut: tokenAddr,
			amountIn: "1", wantErr: ErrZeroAmount.Error()},
		{name: "buy tax floors to zero fee", sim: taxedSim, tokenIn: wethAddr, tokenOut: tokenAddr,
			amountIn: "10", wantOut: "8", wantFee: "0", wantRem: "0", wantGas: 352500},
		{name: "sell tax gate: output taxes to zero", sim: taxedSim, tokenIn: tokenAddr, tokenOut: wethAddr,
			amountIn: "1", wantErr: ErrZeroAmount.Error()},
		{name: "zero amount", sim: taxedSim, tokenIn: wethAddr, tokenOut: tokenAddr, amountIn: "0",
			wantErr: ErrZeroAmount.Error()},
		{name: "negative amount", sim: taxedSim, tokenIn: wethAddr, tokenOut: tokenAddr, amountIn: "-5",
			wantErr: ErrZeroAmount.Error()},
		{name: "invalid pair", sim: taxedSim, tokenIn: tokenAddr,
			tokenOut: "0xdead000000000000000000000000000000000000", amountIn: oneFinney,
			wantErr: ErrInvalidPair.Error()},
		{name: "both counter assets, neither XMA", sim: taxedSim, tokenIn: wethAddr, tokenOut: usdcAddr,
			amountIn: oneFinney, wantErr: ErrInvalidPair.Error()},
		{name: "unknown token in", sim: taxedSim, tokenIn: "0xdead000000000000000000000000000000000000",
			tokenOut: tokenAddr, amountIn: oneFinney, wantErr: ErrInvalidPair.Error()},
		{name: "amount beyond pool liquidity", sim: taxedSim, tokenIn: wethAddr, tokenOut: tokenAddr,
			amountIn: "10000000000000000000000000000000000000000", wantErr: "below smallest"},
		{name: "XMA sell pinned at launch floor", sim: xmaFloorSim, tokenIn: xmaAddr, tokenOut: wethAddr,
			amountIn: "1000000000000000000", wantOut: "23948202847341612", wantFee: "0",
			wantRem: "974430465690135732", wantGas: 350000},
		{name: "XMA buy taxed", sim: xmaTaxedSim, tokenIn: wethAddr, tokenOut: xmaAddr, amountIn: oneFinney,
			wantOut: "939616290878428", wantFee: "50000000000000", wantRem: "0", wantGas: 352500},
		{name: "XMA on both sides", sim: xmaTaxedSim, tokenIn: xmaAddr, tokenOut: xmaAddr, amountIn: oneFinney,
			wantErr: ErrInvalidPair.Error()},
	}
}

// TestCalcAmtOutParityWithLegacy checks the native uint256 CalcAmtOut against the pre-port golden quotes.
func TestCalcAmtOutParityWithLegacy(t *testing.T) {
	for _, c := range parityCases() {
		t.Run(c.name, func(t *testing.T) {
			amountIn, ok := new(big.Int).SetString(c.amountIn, 10)
			require.True(t, ok)
			var in uint256.Int
			if amountIn.Sign() < 0 {
				assertParity(t, c, nil, ErrZeroAmount)
				return
			}
			require.False(t, in.SetFromBig(amountIn))
			res, err := c.sim(t).CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: c.tokenIn, Amount: in},
				TokenOut:      c.tokenOut,
			})
			assertParity(t, c, res, err)
		})
	}
}

func assertParity(t *testing.T, c parityCase, res *pool.CalcAmtOutResult, err error) {
	t.Helper()
	if c.wantErr != "" {
		require.ErrorContains(t, err, c.wantErr)
		return
	}
	require.NoError(t, err)
	assert.Equal(t, c.wantOut, res.TokenAmountOut.Amount.Dec(), "amount out")
	assert.Equal(t, c.wantFee, res.Fee.Amount.Dec(), "fee")
	assert.Equal(t, c.wantRem, res.RemainingTokenAmountIn.Amount.Dec(), "remaining amount in")
	assert.Equal(t, c.wantGas, res.Gas, "gas")
}
