package lunya

import (
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// quoteFixture is a pool as PoolTracker read it, next to what LunyaQuoter returned at the same block.
// TestQuotesMatchLunyaQuoter writes one per pool when LUNYA_WRITE_FIXTURES is set - and one per fee
// token for the STABLE pool, which was re-read after each setFeeToken.
type quoteFixture struct {
	Pool     entity.Pool `json:"pool"`
	ExactIn  []quoteCase `json:"exactIn"`
	ExactOut []quoteCase `json:"exactOut"`
}

// quoteCase leaves the quoted side empty when the quoter reverted or, for exact output, could not
// deliver the full amount.
type quoteCase struct {
	TokenIn   string `json:"tokenIn"`
	TokenOut  string `json:"tokenOut"`
	AmountIn  string `json:"amountIn"`
	AmountOut string `json:"amountOut"`
}

var quoterABI = lo.Must(abi.JSON(strings.NewReader(`[
  {"type":"function","name":"quoteExactInputSingle","stateMutability":"nonpayable",
   "inputs":[{"name":"tokenIn","type":"address"},{"name":"tokenOut","type":"address"},{"name":"poolType","type":"uint8"},
             {"name":"amountIn","type":"uint256"},{"name":"sqrtPriceLimitX96","type":"uint160"}],
   "outputs":[{"name":"amountOut","type":"uint256"}]},
  {"type":"function","name":"quoteExactOutputSingle","stateMutability":"nonpayable",
   "inputs":[{"name":"tokenIn","type":"address"},{"name":"tokenOut","type":"address"},{"name":"poolType","type":"uint8"},
             {"name":"amountOut","type":"uint256"},{"name":"sqrtPriceLimitX96","type":"uint160"}],
   "outputs":[{"name":"amountIn","type":"uint256"},{"name":"amountOutReceived","type":"uint256"}]}
]`)))

// fractions spans a millionth of a reserve to three times it.
func fractions(reserve *big.Int) []*big.Int {
	var amounts []*big.Int
	for _, f := range [][2]int64{{1, 1_000_000}, {1, 10_000}, {1, 100}, {1, 10}, {1, 2}, {9, 10}, {1, 1}, {3, 1}} {
		amount := new(big.Int).Mul(reserve, big.NewInt(f[0]))
		if amount.Div(amount, big.NewInt(f[1])).Sign() > 0 {
			amounts = append(amounts, amount)
		}
	}
	return amounts
}

func quoteRequest(t *testing.T, blockNumber uint64) *ethrpc.Request {
	return newTestnetClient().NewRequest().SetContext(t.Context()).SetBlockNumber(new(big.Int).SetUint64(blockNumber))
}

func TestQuotesMatchLunyaQuoter(t *testing.T) {
	test.SkipCI(t)

	for _, tc := range []struct {
		name       string
		address    string
		startBlock uint64
	}{
		{"CP pool", testnetCPPool, testnetCPPoolBlock},
		{"CL pool", testnetCLPool, testnetCLPoolBlock},
		{"STABLE pool", testnetStablePool, testnetStablePoolBlock},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, extra := trackTestnetPool(t, discoverTestnetPool(t, tc.startBlock, tc.address))
			sim, err := NewPoolSimulator(p)
			require.NoError(t, err)

			var staticExtra StaticExtra
			require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &staticExtra))

			fixture := quoteFixture{Pool: p}
			for _, zeroForOne := range []bool{true, false} {
				in, out := 0, 1
				if !zeroForOne {
					in, out = 1, 0
				}
				tokenIn, tokenOut := p.Tokens[in].Address, p.Tokens[out].Address
				limit := sim.priceLimit(zeroForOne).ToBig()

				amountsIn := fractions(bignumber.NewBig10(p.Reserves[in]))
				quotedOut := make([]*big.Int, len(amountsIn))
				req := quoteRequest(t, p.BlockNumber)
				for i, amountIn := range amountsIn {
					req.AddCall(&ethrpc.Call{
						ABI: quoterABI, Target: arcTestnetQuoter, Method: "quoteExactInputSingle",
						Params: []any{common.HexToAddress(tokenIn), common.HexToAddress(tokenOut),
							staticExtra.PoolType, amountIn, limit},
					}, []any{&quotedOut[i]})
				}
				resp, err := req.TryAggregate()
				require.NoError(t, err)
				for i, amountIn := range amountsIn {
					c := quoteCase{TokenIn: tokenIn, TokenOut: tokenOut, AmountIn: amountIn.String()}
					if resp.Result[i] {
						c.AmountOut = quotedOut[i].String()
					}
					fixture.ExactIn = append(fixture.ExactIn, c)
				}

				reserveOut := bignumber.NewBig10(p.Reserves[out])
				amountsOut := lo.Filter(fractions(reserveOut), func(a *big.Int, _ int) bool {
					return a.Cmp(reserveOut) < 0
				})
				quotedIn := make([]struct {
					AmountIn          *big.Int
					AmountOutReceived *big.Int
				}, len(amountsOut))
				req = quoteRequest(t, p.BlockNumber)
				for i, amountOut := range amountsOut {
					req.AddCall(&ethrpc.Call{
						ABI: quoterABI, Target: arcTestnetQuoter, Method: "quoteExactOutputSingle",
						Params: []any{common.HexToAddress(tokenIn), common.HexToAddress(tokenOut),
							staticExtra.PoolType, amountOut, limit},
					}, []any{&quotedIn[i]})
				}
				resp, err = req.TryAggregate()
				require.NoError(t, err)
				for i, amountOut := range amountsOut {
					c := quoteCase{TokenIn: tokenIn, TokenOut: tokenOut, AmountOut: amountOut.String()}
					if resp.Result[i] && quotedIn[i].AmountOutReceived.Cmp(amountOut) >= 0 {
						c.AmountIn = quotedIn[i].AmountIn.String()
					}
					fixture.ExactOut = append(fixture.ExactOut, c)
				}
			}

			assertFixture(t, fixture)

			if os.Getenv("LUNYA_WRITE_FIXTURES") != "" {
				bytes, err := json.MarshalIndent(fixture, "", "  ")
				require.NoError(t, err)
				name := p.Address + ".json"
				if staticExtra.PoolType == poolTypeStable {
					name = fmt.Sprintf("%s-feetoken%d.json", p.Address, extra.FeeToken)
				}
				require.NoError(t, os.WriteFile(filepath.Join("testdata", name), bytes, 0o644))
			}
		})
	}
}

func assertFixture(t *testing.T, f quoteFixture) {
	t.Helper()

	sim, err := NewPoolSimulator(f.Pool)
	require.NoError(t, err)

	for _, c := range f.ExactIn {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: c.TokenIn, Amount: bignumber.NewBig10(c.AmountIn)},
			TokenOut:      c.TokenOut,
		})
		if c.AmountOut == "" || c.AmountOut == "0" {
			assert.Error(t, err, "exact in %s %s: the quoter reverted or paid nothing", c.AmountIn, c.TokenIn)
		} else if assert.NoError(t, err, "exact in %s %s", c.AmountIn, c.TokenIn) {
			assert.Equal(t, c.AmountOut, res.TokenAmountOut.Amount.String(), "exact in %s %s", c.AmountIn, c.TokenIn)
		}
	}

	for _, c := range f.ExactOut {
		res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: c.TokenOut, Amount: bignumber.NewBig10(c.AmountOut)},
			TokenIn:        c.TokenIn,
		})
		if c.AmountIn == "" {
			assert.Error(t, err, "exact out %s %s: the quoter could not fill it", c.AmountOut, c.TokenOut)
		} else if assert.NoError(t, err, "exact out %s %s", c.AmountOut, c.TokenOut) {
			assert.Equal(t, c.AmountIn, res.TokenAmountIn.Amount.String(), "exact out %s %s", c.AmountOut, c.TokenOut)
		}
	}
}

func readFixtures(t *testing.T) map[string]quoteFixture {
	t.Helper()

	files, err := filepath.Glob(filepath.Join("testdata", "0x*.json"))
	require.NoError(t, err)
	require.NotEmpty(t, files)

	fixtures := make(map[string]quoteFixture, len(files))
	for _, file := range files {
		bytes, err := os.ReadFile(file)
		require.NoError(t, err)
		var f quoteFixture
		require.NoError(t, json.Unmarshal(bytes, &f))
		fixtures[filepath.Base(file)] = f
	}
	return fixtures
}

func TestCalcAmountMatchesQuoterFixtures(t *testing.T) {
	t.Parallel()

	for name, f := range readFixtures(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			assertFixture(t, f)
		})
	}
}

func TestUpdateBalance(t *testing.T) {
	t.Parallel()

	for name, f := range readFixtures(t) {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			sim, err := NewPoolSimulator(f.Pool)
			require.NoError(t, err)

			for _, zeroForOne := range []bool{true, false} {
				in, out := 0, 1
				if !zeroForOne {
					in, out = 1, 0
				}
				amountIn := new(big.Int).Div(sim.GetReserves()[in], big.NewInt(20))
				params := pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: sim.GetTokens()[in], Amount: amountIn},
					TokenOut:      sim.GetTokens()[out],
				}

				before, err := sim.CalcAmountOut(params)
				require.NoError(t, err)

				clone := sim.CloneState()
				swapped, err := clone.CalcAmountOut(params)
				require.NoError(t, err)
				assert.Equal(t, before.TokenAmountOut.Amount, swapped.TokenAmountOut.Amount, "quoting is pure")

				clone.UpdateBalance(pool.UpdateBalanceParams{
					TokenAmountIn:  params.TokenAmountIn,
					TokenAmountOut: *swapped.TokenAmountOut,
					SwapInfo:       swapped.SwapInfo,
				})

				again, err := sim.CalcAmountOut(params)
				require.NoError(t, err)
				assert.Equal(t, before.TokenAmountOut.Amount, again.TokenAmountOut.Amount, "the original is untouched")

				after, err := clone.CalcAmountOut(params)
				require.NoError(t, err)
				assert.Negative(t, after.TokenAmountOut.Amount.Cmp(before.TokenAmountOut.Amount),
					"the clone prices after its own swap")
				assert.Equal(t, new(big.Int).Add(sim.GetReserves()[in], amountIn), clone.GetReserves()[in])
			}
		})
	}
}
