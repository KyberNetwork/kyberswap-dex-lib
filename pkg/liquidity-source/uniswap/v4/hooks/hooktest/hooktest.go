// Package hooktest builds uniswap-v4 pool simulators around a hook for hook tests.
package hooktest

import (
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// NewPoolSimulator returns a full-range 0.3% pool (18-decimal tokens, ~2e10 token0 per token1) whose swaps go
// through the hook registered at hookAddress, tracked as hookExtra.
func NewPoolSimulator(t testing.TB, hookAddress common.Address, hookExtra any) *uniswapv4.PoolSimulator {
	t.Helper()
	return NewPoolSimulatorAt(t, hookAddress, hookExtra, 0)
}

// NewPoolSimulatorAt is NewPoolSimulator with a pool Timestamp, so an indexing build prices at
// that snapshot instead of the wall clock.
func NewPoolSimulatorAt(t testing.TB, hookAddress common.Address, hookExtra any,
	timestamp int64) *uniswapv4.PoolSimulator {
	t.Helper()
	hX, err := json.Marshal(hookExtra)
	require.NoError(t, err)
	extra, err := json.Marshal(map[string]any{
		"liquidity":    json.RawMessage("596800550298755887846132"),
		"sqrtPriceX96": json.RawMessage("566274760763156353397804"),
		"tickSpacing":  200,
		"tick":         -236988,
		"ticks": json.RawMessage(`[{"index":-887200,"liquidityGross":596800550298755887846132,` +
			`"liquidityNet":596800550298755887846132},{"index":887200,"liquidityGross":596800550298755887846132,` +
			`"liquidityNet":-596800550298755887846132}]`),
		"hX": json.RawMessage(hX),
	})
	require.NoError(t, err)
	staticExtra, err := json.Marshal(map[string]any{
		"0x0":   []bool{false, false},
		"fee":   3000,
		"tS":    200,
		"hooks": hexutil.Encode(hookAddress[:]),
	})
	require.NoError(t, err)

	pSim, err := uniswapv4.NewPoolSimulator(entity.Pool{
		Address:   "0xabb949ef8d1e86c37e2620de318da773273460a843a623c3c05d2b85dc893e77",
		Timestamp: timestamp,
		SwapFee:   3000,
		Exchange:  "uniswap-v4",
		Type:      "uniswap-v4",
		Reserves:  []string{"83499061346031704444327158001", "4265567673400257806"},
		Tokens: []*entity.PoolToken{
			{Address: "0x109ddc73b46b2f8141880a5573d2b0d2acf10b07", Decimals: 18, Swappable: true},
			{Address: "0x4200000000000000000000000000000000000006", Decimals: 18, Swappable: true},
		},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
	}, valueobject.ChainIDBase)
	require.NoError(t, err)
	return pSim
}

// RequireRoundTrip requires CalcAmountOut(CalcAmountIn(amountOut)) to give amountOut to within 1e-9, both ways,
// for amountOut at 1e-6, 1e-3 and 1e-1 of the output reserve, tight enough to catch a wrong exact-out fee rule.
// ErrInsufficientBalance from CalcAmountIn is a pool limit: it asserts that error, then exact-in must quote with no error.
func RequireRoundTrip(t testing.TB, pSim *uniswapv4.PoolSimulator) {
	t.Helper()
	tokens, reserves := pSim.GetTokens(), pSim.GetReserves()
	for idxOut := range tokens {
		tokenIn, tokenOut := tokens[1-idxOut], tokens[idxOut]
		for _, frac := range []uint64{1e6, 1e3, 10} {
			var amountOut uint256.Int
			amountOut.Div(&reserves[idxOut], uint256.NewInt(frac))
			resIn, err := pSim.CalcAmtIn(pool.CalcAmtInParams{
				TokenAmountOut: pool.TokenAmt{Token: tokenOut, Amount: amountOut},
				TokenIn:        tokenIn,
			})
			if errors.Is(err, uniswapv3.ErrInsufficientBalance) {
				exactIn, err := pSim.CalcAmtOut(pool.CalcAmtOutParams{
					TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: amountOut},
					TokenOut:      tokenOut,
				})
				require.NoError(t, err, "exact-in of amountOut %s of token%d", &amountOut, idxOut)
				require.False(t, exactIn.TokenAmountOut.Amount.IsZero(), "exact-in of amountOut %s of token%d",
					&amountOut, idxOut)
				continue
			}
			require.NoError(t, err, "amountOut %s of token%d", &amountOut, idxOut)
			resOut, err := pSim.CalcAmtOut(pool.CalcAmtOutParams{
				TokenAmountIn: pool.TokenAmt{Token: tokenIn, Amount: resIn.TokenAmountIn.Amount},
				TokenOut:      tokenOut,
			})
			require.NoError(t, err)
			var diff, tol uint256.Int
			if resOut.TokenAmountOut.Amount.Gt(&amountOut) {
				diff.Sub(&resOut.TokenAmountOut.Amount, &amountOut)
			} else {
				diff.Sub(&amountOut, &resOut.TokenAmountOut.Amount)
			}
			tol.Div(&amountOut, uint256.NewInt(1e9))
			require.LessOrEqual(t, diff.Cmp(&tol), 0,
				"token%d: amountOut %s, amountIn %s gives %s", idxOut, &amountOut, &resIn.TokenAmountIn.Amount,
				&resOut.TokenAmountOut.Amount)
		}
	}
}
