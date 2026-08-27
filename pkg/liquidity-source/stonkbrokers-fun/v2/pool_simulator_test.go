package stonkbrokersfunv2

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	wethPad    = "0xfcd61b25bbf3abd6cf0070d6328e351cc30eec9f"
	moonToken  = "0x391d8735013cc60f7cca0f2ee611a14dc2e66666"
	wethToken  = "0x0bd7d308f8e1639fab988df18a8011f41eacad73"
	ethUsdFeed = "0x78f3556b67e17df817d51ef5a990cdaf09e8d3a9"
)

// buildLiveLaunch176Pool reproduces context/stonkbrokers/output/explorer.md's
// reference fixture as an entity.Pool: WETH V2 pad, on-chain launch id 176,
// getLaunch(176) + the WETH pad's quoteUsdFeed()/ethUsdFeed() wiring, all
// live-verified against the chain's canonical RPC at block 46325622.
func buildLiveLaunch176Pool(t *testing.T) *PoolSimulator {
	t.Helper()

	staticExtra := StaticExtra{
		Pad:               wethPad,
		Lens:              "0x25b5df581f4b2ed450203f375ad8a28b17f115b3",
		LaunchID:          "176",
		IsWethLane:        true,
		QuoteDecimals:     18,
		BufferTaxBps:      9999,
		StartTaxBps:       0,
		DecayPerMinuteBps: 0,
		BufferSecs:        0,
		WindowSecs:        0,
		StartTime:         1_787_691_927,
		Deadline:          1_787_691_927,
		OpenEnded:         true,
		PostTaxBps:        100,
		MaxBuyPpm:         0,
		GradMcapUsd8:      5_000_000_000_000, // $50,000
		LoadedSupply:      "1000000000000000000000000000",
		QuoteUsdFeed:      ethUsdFeed,
		EthUsdFeed:        ethUsdFeed,
	}
	extra := Extra{
		VQuote:       uDec(t, "410349618506110198"),
		VToken:       uDec(t, "999010726104174939447103989"),
		SellsEnabled: true,
		Armed:        true,
		DirectFeed: &OracleReading{
			Answer:    uint256.NewInt(245_794_157_469),
			Decimals:  8,
			UpdatedAt: uint64(time.Now().Unix()), // fresh, so staleness never trips in these tests
			Ok:        true,
		},
	}

	seBytes, err := json.Marshal(staticExtra)
	require.NoError(t, err)
	exBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	ep := entity.Pool{
		Address:  wethPad + "#176",
		Exchange: string(DexType),
		Type:     DexType,
		Tokens: []*entity.PoolToken{
			{Address: moonToken, Decimals: 18, Swappable: true},
			{Address: wethToken, Decimals: 18, Swappable: true},
		},
		Reserves:    entity.PoolReserves{extra.VToken.Dec(), extra.VQuote.Dec()},
		StaticExtra: string(seBytes),
		Extra:       string(exBytes),
		BlockNumber: 46_325_622,
	}

	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

func TestCalcAmountOut_LiveVerifiedWethLaunch176(t *testing.T) {
	sim := buildLiveLaunch176Pool(t)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethToken, Amount: big.NewInt(0).SetUint64(10_000_000_000_000_000)},
		TokenOut:      moonToken,
	})
	require.NoError(t, err)
	require.Equal(t, "23534122942428164868379888", res.TokenAmountOut.Amount.String())
	require.Equal(t, "100000000000000", res.Fee.Amount.String())
	require.Equal(t, int64(defaultGas), res.Gas)

	swapInfo, ok := res.SwapInfo.(SwapInfo)
	require.True(t, ok)
	require.False(t, swapInfo.Graduates) // $50k grad mcap, this trade doesn't get close
}

func TestCalcAmountOut_SellDirectionRejected(t *testing.T) {
	sim := buildLiveLaunch176Pool(t)
	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: moonToken, Amount: big.NewInt(1_000_000)},
		TokenOut:      wethToken,
	})
	require.ErrorIs(t, err, ErrSellNotSupported)
}

func TestCalcAmountOut_GatesRejectUntradeableStates(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*PoolSimulator)
		wantErr error
	}{
		{"aborted", func(s *PoolSimulator) { s.extra.Aborted = true }, ErrPoolAborted},
		{"bonded", func(s *PoolSimulator) { s.extra.Bonded = true }, ErrPoolBonded},
		{"graduated", func(s *PoolSimulator) { s.extra.Graduated = true }, ErrPoolGraduated},
		{"not armed", func(s *PoolSimulator) { s.extra.Armed = false }, ErrPoolNotArmed},
		{
			"window closed (not open-ended, deadline passed)",
			func(s *PoolSimulator) { s.staticExtra.OpenEnded = false; s.staticExtra.Deadline = 1 },
			ErrWindowClosed,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			sim := buildLiveLaunch176Pool(t)
			tc.mutate(sim)
			_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
				TokenAmountIn: pool.TokenAmount{Token: wethToken, Amount: big.NewInt(1_000_000_000_000_000)},
				TokenOut:      moonToken,
			})
			require.ErrorIs(t, err, tc.wantErr)
		})
	}
}

// TestCalcAmountOut_StalePriceRejected is the concrete enforcement test for
// the coordinator's requirement: a stale oracle must reject the BUY quote,
// not silently price it. 49 hours old exceeds SafeLaunchTwapLib's 48h window.
func TestCalcAmountOut_StalePriceRejected(t *testing.T) {
	sim := buildLiveLaunch176Pool(t)
	sim.extra.DirectFeed.UpdatedAt = uint64(time.Now().Add(-49 * time.Hour).Unix())

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethToken, Amount: big.NewInt(1_000_000_000_000_000)},
		TokenOut:      moonToken,
	})
	require.ErrorIs(t, err, ErrStalePrice)
}

func TestCalcAmountOut_TwapModeMissingReadingRejected(t *testing.T) {
	sim := buildLiveLaunch176Pool(t)
	// Simulate a TWAP-mode lane (StaticExtra.TwapPool set) whose tracker read
	// failed -- must reject rather than quote with a nil Twap snapshot.
	sim.staticExtra.QuoteUsdFeed = ""
	sim.staticExtra.TwapPool = "0x52e65b17fb6e5ba00ed806f37afcd2daa50271ca"
	sim.extra.DirectFeed = nil
	sim.extra.Twap = nil

	_, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethToken, Amount: big.NewInt(1_000_000_000_000_000)},
		TokenOut:      moonToken,
	})
	require.ErrorIs(t, err, ErrBadOracleAnswer)
}

func TestUpdateBalance_ConsumesSwapInfo(t *testing.T) {
	sim := buildLiveLaunch176Pool(t)
	origVQuote := new(uint256.Int).Set(sim.extra.VQuote)
	origVToken := new(uint256.Int).Set(sim.extra.VToken)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethToken, Amount: big.NewInt(0).SetUint64(10_000_000_000_000_000)},
		TokenOut:      moonToken,
	})
	require.NoError(t, err)

	// CalcAmountOut must not have mutated state.
	require.True(t, sim.extra.VQuote.Eq(origVQuote))
	require.True(t, sim.extra.VToken.Eq(origVToken))

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: wethToken, Amount: big.NewInt(0).SetUint64(10_000_000_000_000_000)},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})

	swapInfo := res.SwapInfo.(SwapInfo)
	require.True(t, sim.extra.VQuote.Eq(swapInfo.NewVQuote))
	require.True(t, sim.extra.VToken.Eq(swapInfo.NewVToken))
	require.False(t, sim.extra.VQuote.Eq(origVQuote))
}

func TestCloneState_IsIndependent(t *testing.T) {
	sim := buildLiveLaunch176Pool(t)
	clone := sim.CloneState().(*PoolSimulator)

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethToken, Amount: big.NewInt(0).SetUint64(10_000_000_000_000_000)},
		TokenOut:      moonToken,
	})
	require.NoError(t, err)
	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: wethToken, Amount: big.NewInt(0).SetUint64(10_000_000_000_000_000)},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})

	require.False(t, clone.extra.VQuote.Eq(sim.extra.VQuote), "clone must not observe the source's post-swap state")
	require.Equal(t, "410349618506110198", clone.extra.VQuote.Dec())
}

func TestGetMetaInfo_ReturnsPinnedBlockNumberAndCallTarget(t *testing.T) {
	sim := buildLiveLaunch176Pool(t)
	meta, ok := sim.GetMetaInfo("", "").(PoolMeta)
	require.True(t, ok)
	require.Equal(t, uint64(46_325_622), meta.BlockNumber)
	require.Equal(t, wethPad, meta.Pad)
	require.Equal(t, "176", meta.LaunchID)
	require.Equal(t, wethPad, meta.ApprovalAddress)
}

// TestUpdateBalance_MatchesRealStateTransition pins UpdateBalance against a real
// buyEth(176, ...) executed on a Tenderly fork of Robinhood Chain at block
// 46382449 (tx 0x4beb3795102578346ab5a5f688434e75cba2101a36382377e500b95764af6235).
//
// The fork's pre-state is buildLiveLaunch176Pool's fixture. After sending
// 0.01 ETH the contract emitted a Transfer of exactly tokensOut and getLaunch(176)
// reported the vQuote/vToken below, so these are the chain's own numbers rather
// than a restatement of what the simulator computed. Together with
// TestCalcAmountOut_LiveVerifiedWethLaunch176 (which pins the quote itself) this
// closes the loop: quote, emitted transfer, and post-trade reserves all agree.
func TestUpdateBalance_MatchesRealStateTransition(t *testing.T) {
	const (
		realTokensOut = "23534122942428164868379888"
		realVQuote    = "420249618506110198"
		realVToken    = "975476603161746774578724101"
	)

	sim := buildLiveLaunch176Pool(t)
	amountIn := big.NewInt(0).SetUint64(10_000_000_000_000_000) // 0.01 ETH

	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethToken, Amount: amountIn},
		TokenOut:      moonToken,
	})
	require.NoError(t, err)
	require.Equal(t, realTokensOut, res.TokenAmountOut.Amount.String(),
		"quote must match the ERC20 Transfer the pad actually emitted")

	sim.UpdateBalance(pool.UpdateBalanceParams{
		TokenAmountIn:  pool.TokenAmount{Token: wethToken, Amount: amountIn},
		TokenAmountOut: *res.TokenAmountOut,
		SwapInfo:       res.SwapInfo,
	})

	require.Equal(t, realVQuote, sim.extra.VQuote.Dec(),
		"post-swap vQuote must match getLaunch(176) after the real buy")
	require.Equal(t, realVToken, sim.extra.VToken.Dec(),
		"post-swap vToken must match getLaunch(176) after the real buy")
}

// buildPoolWithModes clones the launch-176 fixture and overrides the two
// LaunchModes flags that decide whether an aggregator route can execute at all.
func buildPoolWithModes(t *testing.T, eoaOnly bool, maxBuyPpm uint32) *PoolSimulator {
	t.Helper()
	sim := buildLiveLaunch176Pool(t)
	sim.staticExtra.EoaOnly = eoaOnly
	sim.staticExtra.MaxBuyPpm = maxBuyPpm
	return sim
}

func quoteBuy176(t *testing.T, sim *PoolSimulator, amountIn *big.Int) (*pool.CalcAmountOutResult, error) {
	t.Helper()
	return sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: wethToken, Amount: amountIn},
		TokenOut:      moonToken,
	})
}

// TestCalcAmountOut_EoaOnlyRejected: _tradeGates reverts NotEoa() whenever
// eoaOnly is set and msg.sender != tx.origin. A route always calls buy() from
// the executor contract, so such a launch can never execute and must not be
// quoted. 74 of 283 live launches set this flag.
func TestCalcAmountOut_EoaOnlyRejected(t *testing.T) {
	_, err := quoteBuy176(t, buildPoolWithModes(t, true, 0), big.NewInt(10_000_000_000_000_000))
	require.ErrorIs(t, err, ErrEoaOnly)

	// The same launch without the flag still quotes.
	res, err := quoteBuy176(t, buildPoolWithModes(t, false, 0), big.NewInt(10_000_000_000_000_000))
	require.NoError(t, err)
	require.Equal(t, "23534122942428164868379888", res.TokenAmountOut.Amount.String())
}

// TestCalcAmountOut_BuyCapRejected pins the single-trade upper bound of
// _buy's BuyCapExceeded() check against the numbers measured on-chain: launch
// #174 (loadedSupply 1e27, maxBuyPpm 20000 -> cap 2e25) reverts for a 0.01
// quote-asset buy, while #120's larger 4.2069% cap does not.
func TestCalcAmountOut_BuyCapRejected(t *testing.T) {
	const amount = 10_000_000_000_000_000 // 0.01, quotes ~2.3534e25 tokens

	// cap 2e25 < tokensOut -> refused, matching the on-chain BuyCapExceeded().
	_, err := quoteBuy176(t, buildPoolWithModes(t, false, 20_000), big.NewInt(amount))
	require.ErrorIs(t, err, ErrBuyCapExceeded)

	// cap 4.2069e25 > tokensOut -> allowed, matching the on-chain success.
	res, err := quoteBuy176(t, buildPoolWithModes(t, false, 42_069), big.NewInt(amount))
	require.NoError(t, err)
	require.Equal(t, "23534122942428164868379888", res.TokenAmountOut.Amount.String())

	// A cap of zero means "no cap" on-chain, never "cap of nothing".
	res, err = quoteBuy176(t, buildPoolWithModes(t, false, 0), big.NewInt(amount))
	require.NoError(t, err)
	require.Equal(t, "23534122942428164868379888", res.TokenAmountOut.Amount.String())
}

// TestCalcAmountOut_BuyCapBoundary walks the cap boundary exactly. _buy uses
// `boughtOf > cap` (strictly greater reverts), so a tokensOut equal to the cap
// must be allowed and one wei more must not. The cap itself is computed the way
// the contract computes it -- floor(loadedSupply*ppm/1e6) -- rather than assumed.
func TestCalcAmountOut_BuyCapBoundary(t *testing.T) {
	const amount = 10_000_000_000_000_000

	res, err := quoteBuy176(t, buildPoolWithModes(t, false, 0), big.NewInt(amount))
	require.NoError(t, err)
	out := res.TokenAmountOut.Amount
	supply := buildLiveLaunch176Pool(t).loadedSupply.ToBig()

	capFor := func(ppm uint32) *big.Int {
		c := new(big.Int).Mul(supply, big.NewInt(int64(ppm)))
		return c.Div(c, big.NewInt(1_000_000))
	}

	// Smallest ppm whose cap reaches tokensOut (ceil division).
	ppm := new(big.Int).Add(new(big.Int).Mul(out, big.NewInt(1_000_000)), new(big.Int).Sub(supply, big.NewInt(1)))
	ppm.Div(ppm, supply)
	atCap := uint32(ppm.Uint64())

	require.GreaterOrEqual(t, capFor(atCap).Cmp(out), 0, "test setup: cap must reach tokensOut")
	_, err = quoteBuy176(t, buildPoolWithModes(t, false, atCap), big.NewInt(amount))
	require.NoError(t, err, "tokensOut <= cap must be allowed; _buy only reverts on >")

	require.Less(t, capFor(atCap-1).Cmp(out), 0, "test setup: one ppm lower must fall under tokensOut")
	_, err = quoteBuy176(t, buildPoolWithModes(t, false, atCap-1), big.NewInt(amount))
	require.ErrorIs(t, err, ErrBuyCapExceeded)
}
