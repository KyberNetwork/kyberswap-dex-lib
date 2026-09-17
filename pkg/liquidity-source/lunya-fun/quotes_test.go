package lunyafun

import (
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/poolfactory"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

// quoteFixture is a launch as PoolTracker read it, next to what the launch's own quoteBuy and quoteSell
// answered at the same block. Timestamp is that block's, since the anti-snipe surcharge decays with it.
type quoteFixture struct {
	Pool      entity.Pool `json:"pool"`
	Timestamp uint64      `json:"timestamp"`
	Buys      []quoteCase `json:"buys"`
	Sells     []quoteCase `json:"sells"`
}

type quoteCase struct {
	AmountIn  string `json:"amountIn"`
	AmountOut string `json:"amountOut"`
	Refund    string `json:"refund,omitempty"`
}

func newTestnetClient() *ethrpc.Client {
	return ethrpc.New(arcTestnetRPC).SetMulticallContract(common.HexToAddress(arcTestnetMulticall3))
}

// discoverLaunch runs the backfill pool-service drives over the block a launch was created in.
func discoverLaunch(t *testing.T, startBlock uint64, address string) entity.Pool {
	t.Helper()

	backfiller, err := poolfactory.NewFilterLogsBackfiller(NewPoolFactoryDecoder(testnetConfig()),
		newTestnetClient(), poolfactory.BackfillConfig{
			FactoryAddresses:     []common.Address{common.HexToAddress(arcTestnetFactory)},
			StartBlock:           startBlock,
			MaxBlockRangePerScan: 1,
			Exchange:             DexType,
		})
	require.NoError(t, err)

	pools, _, _, err := backfiller.Backfill(t.Context(), nil)
	require.NoError(t, err)

	p, found := lo.Find(pools, func(p entity.Pool) bool { return p.Address == address })
	require.True(t, found, "launch %s not discovered", address)
	return p
}

func trackLaunch(t *testing.T, p entity.Pool) (entity.Pool, Extra) {
	t.Helper()

	tracker, err := NewPoolTracker(testnetConfig(), newTestnetClient())
	require.NoError(t, err)

	p, err = tracker.GetNewPoolState(t.Context(), p, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	return p, extra
}

func blockTimestamp(t *testing.T, blockNumber uint64) uint64 {
	t.Helper()

	header, err := newTestnetClient().GetETHClient().HeaderByNumber(t.Context(),
		new(big.Int).SetUint64(blockNumber))
	require.NoError(t, err)
	return header.Time
}

func TestPoolTracker_ArcTestnet(t *testing.T) {
	test.SkipCI(t)

	t.Run("a launch on its curve", func(t *testing.T) {
		p, extra := trackLaunch(t, discoverLaunch(t, testnetLaunchBlock, testnetLaunch))

		assert.EqualValues(t, phaseTrading, extra.Phase)
		assert.False(t, extra.Reserve.IsZero())
		assert.False(t, extra.Sold.IsZero())
		assert.Positive(t, extra.CurveFeeBps)
		assert.Positive(t, p.BlockNumber)

		// the curve's own terms, fixed when the launch was created
		assert.False(t, extra.VirtualQuote.IsZero())
		assert.True(t, extra.VirtualToken.Gt(extra.CurveSupply))
		assert.Positive(t, extra.OpenedAt)

		// the reserves are what each side can still pay out
		assert.Equal(t, extra.Reserve.Dec(), p.Reserves[0])
		var left big.Int
		left.Sub(extra.CurveSupply.ToBig(), extra.Sold.ToBig())
		assert.Equal(t, left.String(), p.Reserves[1])
	})

	t.Run("a graduated launch stops trading", func(t *testing.T) {
		p, extra := trackLaunch(t, discoverLaunch(t, testnetGraduatedBlock, testnetGraduated))
		assert.EqualValues(t, phaseGraduated, extra.Phase)

		sim, err := NewPoolSimulator(p)
		require.NoError(t, err)
		_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: p.Tokens[0].Address, Amount: big.NewInt(1_000_000)},
			TokenOut:      p.Tokens[1].Address,
		})
		assert.ErrorIs(t, err, ErrNotTrading)
	})
}

// TestQuotesMatchLaunchViews prices swaps against the launch's own quoteBuy and quoteSell at the block
// the tracker read, which is the same arithmetic buy and sell run on-chain.
func TestQuotesMatchLaunchViews(t *testing.T) {
	test.SkipCI(t)

	for _, tc := range []struct {
		name       string
		address    string
		startBlock uint64
	}{
		{"launch", testnetLaunch, testnetLaunchBlock},
		{"second launch", testnetLaunch2, testnetLaunch2Block},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p, extra := trackLaunch(t, discoverLaunch(t, tc.startBlock, tc.address))
			timestamp := blockTimestamp(t, p.BlockNumber)

			fixture := quoteFixture{Pool: p, Timestamp: timestamp}

			// buys, as fractions of the quote the curve already holds
			amountsIn := fractions(extra.Reserve.ToBig())
			quotedBuys := make([]struct {
				TokensOut *big.Int
				Fee       *big.Int
				Refund    *big.Int
			}, len(amountsIn))
			req := newTestnetClient().NewRequest().SetContext(t.Context()).
				SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber))
			for i, amountIn := range amountsIn {
				req.AddCall(&ethrpc.Call{
					ABI: launchABI, Target: p.Address, Method: "quoteBuy", Params: []any{amountIn},
				}, []any{&quotedBuys[i]})
			}
			resp, err := req.TryAggregate()
			require.NoError(t, err)
			for i, amountIn := range amountsIn {
				require.True(t, resp.Result[i], "quoteBuy reverted for %s", amountIn)
				fixture.Buys = append(fixture.Buys, quoteCase{
					AmountIn:  amountIn.String(),
					AmountOut: quotedBuys[i].TokensOut.String(),
					Refund:    quotedBuys[i].Refund.String(),
				})
			}

			// sells, as fractions of what the curve has sold
			tokensIn := fractions(extra.Sold.ToBig())
			quotedSells := make([]struct {
				AmountOut *big.Int
				Fee       *big.Int
			}, len(tokensIn))
			req = newTestnetClient().NewRequest().SetContext(t.Context()).
				SetBlockNumber(new(big.Int).SetUint64(p.BlockNumber))
			for i, amount := range tokensIn {
				req.AddCall(&ethrpc.Call{
					ABI: launchABI, Target: p.Address, Method: "quoteSell", Params: []any{amount},
				}, []any{&quotedSells[i]})
			}
			resp, err = req.TryAggregate()
			require.NoError(t, err)
			for i, amount := range tokensIn {
				require.True(t, resp.Result[i], "quoteSell reverted for %s", amount)
				fixture.Sells = append(fixture.Sells, quoteCase{
					AmountIn:  amount.String(),
					AmountOut: quotedSells[i].AmountOut.String(),
				})
			}

			assertFixture(t, fixture)

			if os.Getenv("LUNYA_WRITE_FIXTURES") != "" {
				bytes, err := json.MarshalIndent(fixture, "", "  ")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join("testdata", p.Address+".json"), bytes, 0o644))
			}
		})
	}
}

// fractions spans a millionth of an amount to three times it.
func fractions(amount *big.Int) []*big.Int {
	var amounts []*big.Int
	for _, f := range [][2]int64{{1, 1_000_000}, {1, 10_000}, {1, 100}, {1, 10}, {1, 2}, {1, 1}, {3, 1}} {
		scaled := new(big.Int).Mul(amount, big.NewInt(f[0]))
		if scaled.Div(scaled, big.NewInt(f[1])).Sign() > 0 {
			amounts = append(amounts, scaled)
		}
	}
	return amounts
}

// assertFixture prices every case with the clock the quotes were taken at, since the surcharge decays
// with time.
func assertFixture(t *testing.T, f quoteFixture) {
	t.Helper()

	previous := nowUnix
	nowUnix = func() uint64 { return f.Timestamp }
	defer func() { nowUnix = previous }()

	sim, err := NewPoolSimulator(f.Pool)
	require.NoError(t, err)
	quote, base := f.Pool.Tokens[0].Address, f.Pool.Tokens[1].Address

	for _, c := range f.Buys {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: quote, Amount: bignumber.NewBig10(c.AmountIn)},
			TokenOut:      base,
		})
		if c.AmountOut == "0" {
			assert.Error(t, err, "buy %s: the curve pays nothing", c.AmountIn)
			continue
		}
		if assert.NoError(t, err, "buy %s", c.AmountIn) {
			assert.Equal(t, c.AmountOut, res.TokenAmountOut.Amount.String(), "buy %s", c.AmountIn)
			assert.Equal(t, c.Refund, res.RemainingTokenAmountIn.Amount.String(), "buy %s refund", c.AmountIn)
		}
	}

	for _, c := range f.Sells {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: base, Amount: bignumber.NewBig10(c.AmountIn)},
			TokenOut:      quote,
		})
		if c.AmountOut == "0" {
			assert.Error(t, err, "sell %s: the curve pays nothing", c.AmountIn)
			continue
		}
		if assert.NoError(t, err, "sell %s", c.AmountIn) {
			assert.Equal(t, c.AmountOut, res.TokenAmountOut.Amount.String(), "sell %s", c.AmountIn)
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

func TestCalcAmountMatchesLaunchFixtures(t *testing.T) {
	for name, f := range readFixtures(t) {
		t.Run(name, func(t *testing.T) {
			assertFixture(t, f)
		})
	}
}

func TestUpdateBalance(t *testing.T) {
	for name, f := range readFixtures(t) {
		t.Run(name, func(t *testing.T) {
			previous := nowUnix
			nowUnix = func() uint64 { return f.Timestamp }
			defer func() { nowUnix = previous }()

			sim, err := NewPoolSimulator(f.Pool)
			require.NoError(t, err)
			quote, base := f.Pool.Tokens[0].Address, f.Pool.Tokens[1].Address

			amountIn := new(big.Int).Div(sim.GetReserves()[0], big.NewInt(10))
			params := pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: quote, Amount: amountIn},
				TokenOut:      base,
			}

			before, err := sim.CalcAmountOut(params)
			require.NoError(t, err)

			clone := sim.CloneState()
			bought, err := clone.CalcAmountOut(params)
			require.NoError(t, err)
			assert.Equal(t, before.TokenAmountOut.Amount, bought.TokenAmountOut.Amount, "quoting is pure")

			clone.UpdateBalance(pool.UpdateBalanceParams{
				TokenAmountIn:  params.TokenAmountIn,
				TokenAmountOut: *bought.TokenAmountOut,
				SwapInfo:       bought.SwapInfo,
			})

			again, err := sim.CalcAmountOut(params)
			require.NoError(t, err)
			assert.Equal(t, before.TokenAmountOut.Amount, again.TokenAmountOut.Amount, "the original is untouched")

			after, err := clone.CalcAmountOut(params)
			require.NoError(t, err)
			assert.Negative(t, after.TokenAmountOut.Amount.Cmp(before.TokenAmountOut.Amount),
				"the clone prices after its own buy, further along the curve")
		})
	}
}
