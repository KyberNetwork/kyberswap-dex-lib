package lotflow

import (
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const mqDir = "testdata/mq"

func loadMq(t *testing.T) []*Case {
	t.Helper()
	idx, err := LoadIndex(filepath.Join(mqDir, "index.json"))
	require.NoError(t, err)
	var out []*Case
	for _, e := range idx.Cases {
		c, err := LoadCase(filepath.Join(mqDir, filepath.Base(e.File)))
		require.NoError(t, err, e.ID)
		require.NotNil(t, c.Pre.Mq, e.ID)
		out = append(out, c)
	}
	require.Len(t, out, 14)
	return out
}

func twapStateOf(tw *MqTwapInfo) *TwapState {
	s := &TwapState{Tick: tw.Tick, Index: tw.ObservationIndex, Cardinality: tw.ObservationCardinality,
		Obs: map[int]Observation{}}
	for _, o := range tw.ObservationsRead {
		s.Obs[o.Index] = Observation{Timestamp: o.BlockTimestamp.Uint64(), TickCumulative: o.TickCumulative.Copy(),
			Initialized: o.Initialized}
	}
	return s
}

// v3 Oracle.observe([1800, 0]) reproduced from the observation-ring slots the fixture dumped, at the
// swap's block timestamp: both tickCumulatives and the mean tick (rounded toward -inf as MqPricing),
// for the conversion pool and both token/WETH pools of all 14 fixtures (the binary search reads only
// slots the fixture recorded; a missing one is an error, so this also proves the dump is complete).
func TestMqObserve(t *testing.T) {
	n := 0
	for _, c := range loadMq(t) {
		ts := c.Pin.BlockTimestamp.Uint64()
		for _, tw := range c.Pre.Mq.Twap {
			s := twapStateOf(&tw)
			c0, err := s.Observe(ts, uint64(c.Pre.Mq.TwapWindow))
			require.NoError(t, err, c.ID, tw.Pool)
			c1, err := s.Observe(ts, 0)
			require.NoError(t, err, c.ID, tw.Pool)
			eq(t, tw.TickCumulatives[0].Copy(), c0, c.ID, tw.Pool, "cum[window]")
			eq(t, tw.TickCumulatives[1].Copy(), c1, c.ID, tw.Pool, "cum[0]")
			tick, err := s.TwapTick(ts, uint64(c.Pre.Mq.TwapWindow))
			require.NoError(t, err)
			require.Equal(t, tw.TwapTick, tick, c.ID, tw.Pool)
			n++
		}
	}
	require.Equal(t, 42, n)
	t.Logf("MQ observe: %d/42 pool TWAPs exact (both cumulatives + mean tick)", n)
}

// MqPricing.tokenUsdg18 for PIPEDOG and WALLET and NavGuard#2's checked NAV, from TWAPs + feeds alone.
func TestMqPricingAndNav(t *testing.T) {
	spots, navs := 0, 0
	for _, c := range loadMq(t) {
		m := marketOf(t, c)
		for i := range m.Constituents {
			mc := &m.Constituents[i]
			if mc.Mq == nil {
				continue
			}
			sp, err := MqTokenUsdg18(m.Mq, mc.Mq, m.Timestamp)
			require.NoError(t, err, c.ID)
			want := c.Pre.Mq.SpotUsd18[c.Pre.Constituents[i].Symbol]
			eq(t, want.Copy(), sp, c.ID, c.Pre.Constituents[i].Symbol)
			eq(t, c.Pre.Constituents[i].Nav.SpotUsd18.Copy(), sp, c.ID)
			spots++
		}
		res, err := Quote(fixtureConfig(c, FinalisedRules), m, inputOf(c))
		require.NoError(t, err, c.ID)
		eq(t, c.Pre.Nav.CheckedNavPerUnit18.Copy(), res.Nav, c.ID, "checked NAV")
		navs++
	}
	t.Logf("MQ pricing: %d/28 WETH-token spots exact, %d/14 checked NAVs exact", spots, navs)
}

// A6 acceptance: all 14 MQ fixtures (buys/sells $100/$1k/$10k, quote-vs-execute drift Δ 12/60/600 s)
// raw-unit exact: amountOut, treasury, every LegExecuted (kind-3 two-hop legs single-candidate and
// unprobed, memo sells, NVDA probed). The drift fixtures also reproduce the harness's quote at t.
func TestMqWholeSwap(t *testing.T) {
	cases := loadMq(t)
	byID := map[string]*Case{}
	for _, c := range cases {
		byID[c.ID] = c
	}
	drift := 0
	for _, c := range cases {
		res, err := quoteChecked(t, c, marketOf(t, c))
		require.NoError(t, err, c.ID)
		if q := c.Input.A6Info(); q != nil {
			// the same trade at t is the "-t0" fixture of the same run: the simulator's quote there equals
			// the harness's quoteAtT, and the drift is the simulator's two quotes apart
			base := byID[strings.Split(c.ID, "-dt")[0]+"-t0"]
			require.NotNil(t, base, c.ID)
			r0, err := Quote(fixtureConfig(base, FinalisedRules), marketOf(t, base), inputOf(base))
			require.NoError(t, err)
			eq(t, q.QuoteAtT.AmountOut.Copy(), r0.AmountOut, c.ID, "quote at t")
			bps := new(big.Int).Mul(new(big.Int).Sub(res.AmountOut, r0.AmountOut), bigBps)
			bps.Quo(bps, r0.AmountOut)
			require.Equal(t, fmt.Sprint(int64(q.DriftBps)), bps.String(), c.ID, "drift bps")
			drift++
		}
	}
	require.Equal(t, 6, drift)
	t.Logf("MQ whole swap: 14/14 exact; %d drift cases reproduce quote-at-t and drift bps", drift)
}

// MqPricing refusals the fixtures do not reach, from the contract arithmetic on a real pre-state:
// Chainlink ETH/USDG more than 300 bps from the conversion TWAP (exactly 300 passes), a feed older
// than MAX_FEED_AGE, a feed updated in the future. All revert in beforeSwap (NAV is read there).
func TestMqPricingRefusals(t *testing.T) {
	c := loadMq(t)[0]
	cfg := fixtureConfig(c, FinalisedRules)
	m := marketOf(t, c)
	conv, err := m.Mq.ConversionTwapUsdg18(m.Timestamp)
	require.NoError(t, err)
	usdg := m.Mq.UsdgUsd.Answer // 8 decimals
	// pick ETH/USD so ref = conv*(1 +- 300bps) at the boundary: ref = eth*1e18/usdg (both 8 dec)
	ethFor := func(ref *big.Int) *big.Int { return mulDiv(ref, usdg, bigE18) }
	at := func(bps int64, up bool) *big.Int {
		f := big.NewInt(10_000 + bps)
		if !up {
			f = big.NewInt(10_000 - bps)
		}
		return mulDiv(conv, f, bigBps)
	}
	var pcf *MqError
	m.Mq.EthUsd.Answer = ethFor(at(320, true))
	_, err = Quote(cfg, m, inputOf(c))
	require.ErrorAs(t, err, &pcf)
	require.Equal(t, "PriceCheckFailed", pcf.Name)
	require.Equal(t, Selector(err), selectorOf("PriceCheckFailed(uint256,uint256)"))
	var se *SwapError
	require.True(t, errors.As(err, &se))
	require.Equal(t, PhaseBeforeSwap, se.Phase)

	m = marketOf(t, c)
	m.Mq.EthUsd.UpdatedAt = m.Timestamp - 90_001
	_, err = Quote(cfg, m, inputOf(c))
	require.ErrorAs(t, err, &pcf)
	require.Equal(t, "StaleFeed", pcf.Name)
	m.Mq.EthUsd.UpdatedAt = m.Timestamp - 90_000 // exactly MAX_FEED_AGE: accepted
	_, err = Quote(cfg, m, inputOf(c))
	require.NoError(t, err)

	m = marketOf(t, c)
	m.Mq.UsdgUsd.UpdatedAt = m.Timestamp + 1
	_, err = Quote(cfg, m, inputOf(c))
	require.ErrorAs(t, err, &pcf)
	require.Equal(t, "FeedInFuture", pcf.Name)
}

func selectorOf(sig string) string { return Selector(&MqError{Name: "x", sig: sig}) }

func twapFor(c *Case, pool string) *TwapState {
	for i := range c.Pre.Mq.Twap {
		if strings.EqualFold(c.Pre.Mq.Twap[i].Pool, pool) {
			return twapStateOf(&c.Pre.Mq.Twap[i])
		}
	}
	return nil
}

func mqMarketOf(c *Case) *MqMarket {
	q := c.Pre.Mq
	conv := legPoolOf(&q.Conversion)
	conv.Kind = KindV3
	feed := func(f MqFeedInfo) *FeedState {
		return &FeedState{Answer: f.FeedAnswer.Copy(), Decimals: f.FeedDecimals, UpdatedAt: f.FeedUpdatedAt.Uint64()}
	}
	return &MqMarket{Weth: q.Weth, ConvPool: q.ConversionPool, ConvFactory: q.Conversion.Factory, Conv: conv,
		ConvTwap: twapFor(c, q.ConversionPool), EthUsdFeed: q.EthUsdFeed.Address, UsdgUsdFeed: q.UsdgUsdFeed.Address,
		EthUsd: feed(q.EthUsdFeed), UsdgUsd: feed(q.UsdgUsdFeed), UsdgDecimals: c.Pre.Hook.UsdgDecimals,
		Usdg: c.Pre.Hook.Usdg, PmWeth: c.Pre.PM.WethBalance.Copy()}
}
