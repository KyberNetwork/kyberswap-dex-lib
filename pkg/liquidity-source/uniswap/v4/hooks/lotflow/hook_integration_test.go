package lotflow

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"testing"
	"time"

	"github.com/KyberNetwork/ethrpc"
	wire "github.com/KyberNetwork/msgpack/v5"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func integrationHook(t *testing.T, path string) (*Hook, *Case) {
	t.Helper()
	c, err := LoadCase(path)
	require.NoError(t, err)
	h := hookOf(t, c, marketOf(t, c))
	h.TrackedAt = time.Now().Unix()
	return h, c
}

func TestHookIntegrationMsgpackMarket(t *testing.T) {
	for _, path := range []string{"testdata/fixtures/navjit2-buy-1kusd.json", "testdata/mq/mq-buy-1kusd.json"} {
		h, c := integrationHook(t, path)
		if m := h.Market.Mq; m != nil {
			// an RPC-tracked market carries lazy observation loaders; they must not reach the wire
			fetch := func(int) (Observation, error) { return Observation{}, errors.New("not on the wire") }
			m.ConvTwap.Fetch = fetch
			for i := range h.Market.Constituents {
				if q := h.Market.Constituents[i].Mq; q != nil && q.Twap != nil {
					q.Twap.Fetch = fetch
				}
			}
		}
		var buf bytes.Buffer
		en := wire.NewEncoder(&buf)
		en.IncludeUnexported(true)
		en.SetForceAsArray(true)
		require.NoError(t, en.Encode(h.Extra), path)
		de := wire.NewDecoder(&buf)
		de.IncludeUnexported(true)
		de.SetForceAsArray(true)
		var back Extra
		require.NoError(t, de.Decode(&back), path)
		got, err := (&Hook{Extra: back}).BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true,
			ZeroForOne: c.Input.ZeroForOne, AmountSpecified: c.Input.AmountIn.Copy()})
		require.NoError(t, err, path)
		want, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true,
			ZeroForOne: c.Input.ZeroForOne, AmountSpecified: c.Input.AmountIn.Copy()})
		require.NoError(t, err, path)
		require.Equal(t, want.DeltaUnspecified, got.DeltaUnspecified, path)
	}
}

func integrationPool(t *testing.T, h *Hook) entity.Pool {
	t.Helper()
	v := h.Market.Venue
	liquidity := new(big.Int)
	var ticks []uniswapv3.Tick
	if s := v.Standing; s != nil {
		liquidity.Set(s.Liquidity)
		ticks = []uniswapv3.Tick{
			{Index: s.Lower, LiquidityGross: s.Liquidity, LiquidityNet: s.Liquidity},
			{Index: s.Upper, LiquidityGross: s.Liquidity, LiquidityNet: new(big.Int).Neg(s.Liquidity)},
		}
	}
	hookRaw, err := json.Marshal(h.Extra)
	require.NoError(t, err)
	extra, err := json.Marshal(uniswapv4.Extra{Extra: &uniswapv3.Extra{Liquidity: liquidity,
		SqrtPriceX96: v.SqrtPriceX96, Tick: big.NewInt(int64(v.Tick)), TickSpacing: 1, Ticks: ticks}, HookExtra: hookRaw})
	require.NoError(t, err)
	hookAddr := HookAddresses[0]
	if h.Market.Mq != nil {
		hookAddr = HookAddresses[1]
	}
	static, err := json.Marshal(uniswapv4.StaticExtra{Fee: 0, TickSpacing: 1, HooksAddress: hookAddr})
	require.NoError(t, err)
	reserves, err := h.GetReserves(context.Background(), nil)
	require.NoError(t, err)
	tokens := []*entity.PoolToken{{Address: h.Usdg, Decimals: uint8(v.UsdgDecimals), Swappable: true},
		{Address: h.Lot, Decimals: 18, Swappable: true}}
	if !v.UsdgIs0 {
		tokens[0], tokens[1] = tokens[1], tokens[0]
	}
	return entity.Pool{Address: "0x0000000000000000000000000000000000000000000000000000000000000017",
		Exchange: Exchange, Type: uniswapv4.DexType, Tokens: tokens, Reserves: reserves,
		Extra: string(extra), StaticExtra: string(static), BlockNumber: h.Block}
}

// FixturePoolForTest exposes the test-only fixture builder to the external serialization test.
func FixturePoolForTest(t *testing.T, mq bool) entity.Pool {
	t.Helper()
	path := "testdata/a5/amznaapl-buy-1kusd-standing.json"
	if mq {
		path = "testdata/mq/mq-buy-1kusd.json"
	}
	h, _ := integrationHook(t, path)
	return integrationPool(t, h)
}

func TestHookIntegrationPriceLimit(t *testing.T) {
	for _, path := range []string{
		"testdata/fixtures/navjit2-buy-1kusd.json",
		"testdata/a5/amznaapl-buy-1kusd-standing.json",
		"testdata/mq/mq-buy-1kusd.json",
	} {
		t.Run(path, func(t *testing.T) {
			h, _ := integrationHook(t, path)
			provider, ok := any(h).(uniswapv4.HookPriceLimitProvider)
			require.True(t, ok, "JIT hooks must replace standing-tick executor limits")
			sim, err := uniswapv4.NewPoolSimulator(integrationPool(t, h), valueobject.ChainIDRobinhood)
			require.NoError(t, err)
			for _, buy := range []bool{true, false} {
				in, out, amount := h.Usdg, h.Lot, big.NewInt(100_000_000)
				if !buy {
					in, out, amount = h.Lot, h.Usdg, new(big.Int).Mul(big.NewInt(100), bigE18)
				}
				limit := provider.SqrtPriceLimit(buy == h.Market.Venue.UsdgIs0)
				require.NotNil(t, limit)
				meta := sim.GetMetaInfo(in, out).(uniswapv4.PoolMetaInfo)
				require.Equal(t, limit, meta.PriceLimit)
				want, err := QuoteConservative(h.cfg(), h.Market, SwapInput{BuyLot: buy, AmountIn: amount,
					Prefund: true, SqrtPriceLimitX96: meta.PriceLimit.ToBig()})
				require.NoError(t, err)
				require.Zero(t, want.Venue.AmountIn.Cmp(amount), "test trade fills completely at the encoded limit")
				got, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
					TokenAmountIn: pool.TokenAmount{Token: in, Amount: amount}, TokenOut: out})
				require.NoError(t, err)
				require.Zero(t, got.TokenAmountOut.Amount.Cmp(want.AmountOut), "quote uses the executor limit")
				// the complete simulator (outer v4 base gas + hook gas) totals the swap's gas, not +BaseGas
				require.Equal(t, SwapGasFor(gasFeatures(h.Market, buy, want)), got.Gas)
			}
		})
	}
}

func TestHookIntegrationPriceLimitWithinBand(t *testing.T) {
	h, _ := integrationHook(t, "testdata/fixtures/navjit2-buy-1kusd.json")
	nav, err := newMarketState(h.Market).checkedNav()
	require.NoError(t, err)
	for _, buy := range []bool{true, false} {
		limit := h.SqrtPriceLimit(buy)
		require.NotNil(t, limit)
		price, err := h.Market.Venue.PoolUsd18(limit.ToBig())
		require.NoError(t, err)
		difference := new(big.Int).Sub(price, nav)
		require.Equal(t, buy, difference.Sign() > 0)
		difference.Abs(difference).Mul(difference, bigBps)
		offset := h.Market.Venue.Params.BandBps - priceLimitBufferBps
		require.Negative(t, difference.Cmp(new(big.Int).Mul(nav, new(big.Int).SetUint64(offset+1))))
		require.Positive(t, difference.Cmp(new(big.Int).Mul(nav, new(big.Int).SetUint64(offset-1))))
	}
	h.Market.Venue.UsdgIs0 = false
	_, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false,
		AmountSpecified: big.NewInt(1_000_000)})
	require.ErrorIs(t, err, ErrCurrencyOrder)
}

func TestHookIntegrationRefusesPriceLimitedFill(t *testing.T) {
	h, c := integrationHook(t, "testdata/fixtures/amznaapl-buy-maxfill.json")
	_, err := QuoteConservative(h.cfg(), h.Market, inputOf(c))
	require.NoError(t, err, "the unbounded fixture fills")
	_, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true,
		AmountSpecified: c.Input.AmountIn.Copy()})
	require.Error(t, err, "do not quote an unbounded fill when the executor uses the NAV band")
}

// A direction whose spread reaches the buffered band (spread >= bandBps-25) cannot fill inside the
// executor limit; it is refused by name instead of an opaque limit error, and GetReserves reports "0"
// for that side. The live parameter sets (hook #1 25/100/200, hook #2 25/200/600) fill both ways. The
// guard is necessary, not sufficient: a buy edge is the executed cost plus the spread, so a spread just
// inside the bound can still be refused by the limit itself (never quoted as a partial).
func TestHookIntegrationSpreadGuard(t *testing.T) {
	const fill, guard, refuse = "fill", "guard", "refuse"
	for _, tc := range []struct {
		name              string
		buy, sell, band   uint64
		wantBuy, wantSell string
	}{
		{"live hook #1", 25, 100, 200, fill, fill},
		{"live hook #2", 25, 200, 600, fill, fill},
		{"buy spread at limit", 175, 100, 200, guard, fill},
		{"sell spread at limit", 25, 175, 200, fill, guard},
		{"sell spread above limit", 25, 590, 600, fill, guard},
		{"buy spread above band", 250, 100, 200, guard, fill},
		{"buy spread just inside", 174, 100, 200, refuse, fill},
		{"sell spread just inside", 25, 174, 200, fill, refuse},
	} {
		t.Run(tc.name, func(t *testing.T) {
			h, _ := integrationHook(t, "testdata/fixtures/navjit2-buy-1kusd.json")
			p := &h.Market.Venue.Params
			p.BuySpreadBps, p.SellSpreadBps, p.BandBps = tc.buy, tc.sell, tc.band
			v := h.Market.Venue
			for _, buy := range []bool{true, false} {
				amount, want := big.NewInt(100_000_000), tc.wantBuy
				if !buy {
					amount, want = new(big.Int).Mul(big.NewInt(100), bigE18), tc.wantSell
				}
				r, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: v.ZeroForOne(buy),
					AmountSpecified: amount})
				switch want {
				case fill:
					require.NoError(t, err, "buy=%v", buy)
					require.Negative(t, r.DeltaUnspecified.Sign())
					require.NotNil(t, h.SqrtPriceLimit(v.ZeroForOne(buy)))
				case guard:
					require.ErrorIs(t, err, ErrSpreadOutsideLimit, "buy=%v", buy)
					require.Nil(t, h.SqrtPriceLimit(v.ZeroForOne(buy)), "no executor limit advertised")
				case refuse:
					require.Error(t, err, "buy=%v", buy)
					require.NotErrorIs(t, err, ErrSpreadOutsideLimit, "buy=%v", buy)
				}
			}
			reserves, err := h.GetReserves(context.Background(), nil)
			require.NoError(t, err)
			lotRes, usdgRes := reserves[1], reserves[0]
			if !v.UsdgIs0 {
				lotRes, usdgRes = usdgRes, lotRes
			}
			if tc.wantBuy == guard {
				require.Equal(t, "0", lotRes, "LOT side (buys)")
			}
			if tc.wantSell == guard {
				require.Equal(t, "0", usdgRes, "USDG side (sells)")
			}
			if tc.wantBuy == fill {
				require.NotEqual(t, "0", lotRes, "LOT side (buys)")
			}
			if tc.wantSell == fill {
				require.NotEqual(t, "0", usdgRes, "USDG side (sells)")
			}
		})
	}
}

func TestHookIntegrationTrackRefreshesReserves(t *testing.T) {
	h := &Hook{}
	for _, path := range []string{"testdata/fixtures/navjit2-buy-1kusd.json", "testdata/fixtures/mag5-buy-1kusd.json"} {
		c, err := LoadCase(path)
		require.NoError(t, err)
		param := &uniswapv4.HookParam{BlockNumber: big.NewInt(123)}
		raw, err := h.trackMarket(context.Background(), param, trackerOf(c), &fixtureLoader{c}, c.Pre.Lot.Address, time.Now())
		require.NoError(t, err)
		// The outer v4 tracker calls GetReserves on this same instance after Track.
		got, err := h.GetReserves(context.Background(), param)
		require.NoError(t, err)
		fresh := &Hook{}
		require.NoError(t, json.Unmarshal(raw, &fresh.Extra))
		want, err := fresh.GetReserves(context.Background(), param)
		require.NoError(t, err)
		require.NotEqual(t, entity.PoolReserves{"0", "0"}, want)
		require.Equal(t, want, got, "fresh executable reserves must replace empty/previous standing reserves")
		require.Equal(t, c.Pre.Lot.Address, h.Lot)
	}
}

func TestHookIntegrationInvalidNavClearsOldQuotes(t *testing.T) {
	h, c := integrationHook(t, "testdata/fixtures/navjit2-buy-1kusd.json")
	c.Pre.Constituents[0].Nav.FeedAnswer.SetInt64(0)
	param := &uniswapv4.HookParam{BlockNumber: big.NewInt(123)}
	raw, err := h.trackMarket(context.Background(), param, trackerOf(c), &fixtureLoader{c}, c.Pre.Lot.Address, time.Now())
	require.NoError(t, err)
	for _, instance := range []*Hook{h, {}} {
		if instance != h {
			require.NoError(t, json.Unmarshal(raw, &instance.Extra))
		}
		reserves, err := instance.GetReserves(context.Background(), param)
		require.NoError(t, err)
		require.Equal(t, entity.PoolReserves{"0", "0"}, reserves)
		_, err = instance.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true,
			ZeroForOne: true, AmountSpecified: big.NewInt(1_000_000)})
		require.ErrorIs(t, err, ErrNotTracked)
	}
}

func TestHookIntegrationRejectsOverrides(t *testing.T) {
	_, err := (&Hook{}).Track(context.Background(), &uniswapv4.HookParam{
		Overrides: map[common.Address]gethclient.OverrideAccount{{}: {Balance: big.NewInt(1)}}})
	require.ErrorIs(t, err, ErrOverrides, "do not mix overridden outer state with unmodified dependencies")
}

func TestHookIntegrationPinsInitialRPCReads(t *testing.T) {
	v := liveVenues[0]
	tape := loadTape(t, tapePath(v.poolID))
	var ctr rpcCounter
	srv := replayServer(t, tape, &ctr)
	defer srv.Close()
	ctr.reset()
	_, err := (&Hook{}).Track(context.Background(),
		venueParam(ethrpc.New(srv.URL).SetMulticallContract(common.HexToAddress(liveMulticall)), v, nil))
	require.NoError(t, err)
	require.Equal(t, 1, ctr.Methods["eth_blockNumber"], "the block is resolved once")
	require.Equal(t, map[string]int{tape.Block: ctr.Methods["eth_call"]}, ctr.Tags,
		"every read, wiring included, uses the one resolved snapshot")
}

func TestHookIntegrationMqUpdateBalance(t *testing.T) {
	h, c := integrationHook(t, "testdata/mq/mq-buy-1kusd.json")
	clone := h.CloneState().(*Hook)
	initial := new(big.Int).Set(clone.Market.Mq.Conv.SqrtPriceX96)
	r, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true,
		AmountSpecified: c.Input.AmountIn.Copy()})
	require.NoError(t, err)
	// Replay the executed legs, including the leftover sells, to obtain the conversion pool
	// state independently of World/UpdateBalance. The second basket swap must start there.
	replay := newMarketState(h.Market)
	for _, leg := range r.SwapInfo.(*SwapInfo).res.Legs {
		sources, pools, err := replay.sources(h.cfg(), leg.ConstituentIndex)
		require.NoError(t, err)
		matched := false
		for i, source := range sources {
			if !strings.EqualFold(source.ID, leg.Source.ID) {
				continue
			}
			amount := leg.Paid
			if leg.Buy {
				amount = leg.Got
			}
			result := SimulateLeg(h.cfg(), source, pools[i], leg.Buy, amount)
			require.NoError(t, result.Err)
			require.Zero(t, result.Paid.Cmp(leg.Paid))
			require.Zero(t, result.Got.Cmp(leg.Got))
			replay.apply(leg.ConstituentIndex, source, result)
			matched = true
			break
		}
		require.True(t, matched)
	}
	require.NotZero(t, initial.Cmp(replay.conv.SqrtPriceX96), "trade moves the common conversion pool")
	h.UpdateBalance(r.SwapInfo)
	require.Zero(t, h.Market.Mq.Conv.SqrtPriceX96.Cmp(replay.conv.SqrtPriceX96),
		"next quote must use the USDG/WETH pool after the previous basket swap")
	require.Zero(t, clone.Market.Mq.Conv.SqrtPriceX96.Cmp(initial), "the clone remains independent")
}

func TestHookIntegrationConcurrentQuotes(t *testing.T) {
	for _, path := range []string{"testdata/fixtures/navjit2-buy-1kusd.json", "testdata/mq/mq-buy-1kusd.json"} {
		h, c := integrationHook(t, path)
		params := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: c.Input.AmountIn.Copy()}
		before, err := json.Marshal(h.Extra)
		require.NoError(t, err)
		want, err := h.BeforeSwap(params)
		require.NoError(t, err)
		t.Run(path, func(t *testing.T) {
			for i := range 8 {
				t.Run(fmt.Sprint(i), func(t *testing.T) {
					t.Parallel()
					for range 3 {
						clone := h.CloneState().(*Hook)
						got, err := h.BeforeSwap(params)
						require.NoError(t, err)
						require.Equal(t, want.DeltaUnspecified, got.DeltaUnspecified)
						clone.UpdateBalance(got.SwapInfo)
						_, err = clone.BeforeSwap(params)
						require.NoError(t, err)
					}
				})
			}
		})
		after, err := json.Marshal(h.Extra)
		require.NoError(t, err)
		require.Equal(t, before, after, "quotes and updates to clones leave the original snapshot unchanged")
	}
}

func TestHookIntegrationStaleCheck(t *testing.T) {
	h, _ := integrationHook(t, "testdata/fixtures/navjit2-buy-1kusd.json")
	h.TrackedAt = time.Now().Unix() - MaxAgeSec - 1
	for _, stale := range []bool{false, true} {
		sim, err := pool.Factory(uniswapv4.DexType)(pool.FactoryParams{EntityPool: integrationPool(t, h),
			ChainID: valueobject.ChainIDRobinhood, Opts: pool.FactoryOpts{StaleCheck: stale}})
		require.NoError(t, err)
		_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: h.Usdg, Amount: big.NewInt(1_000_000)}, TokenOut: h.Lot})
		if stale {
			require.ErrorIs(t, err, ErrStale)
		} else {
			require.NoError(t, err)
		}
	}
}

// F-N4: Track validates what it publishes, so the Track instance and an instance the factory builds
// from the published extra always agree. Every fixture loaded through Track publishes an extra the
// factory accepts with the same market. The three metal-*-gldHookedOnly fixtures (the keeper left GLD
// only a hooked source; the chain reverts NoExecutableSource) load a constituent with no pools: Track
// publishes the empty market, so both instances refuse, as the chain does.
func TestTrackPublishesOnlyValidExtras(t *testing.T) {
	_, cases := loadAll(t)
	empty := 0
	for _, c := range cases {
		h := &Hook{}
		param := &uniswapv4.HookParam{BlockNumber: big.NewInt(123)}
		raw, err := h.trackMarket(context.Background(), param, trackerOf(c), &fixtureLoader{c}, c.Pre.Lot.Address, time.Now())
		require.NoError(t, err, c.ID)
		require.NoError(t, validateExtra(&h.Extra), c.ID)
		f := hookFromJSON(raw)
		require.True(t, f.Tracked, c.ID)
		require.Equal(t, h.Market == nil, f.Market == nil, c.ID)
		if h.Market == nil {
			empty++
			require.False(t, c.IsFill(), "%s: the chain fills a state Track published as empty", c.ID)
			_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: c.Input.ZeroForOne,
				AmountSpecified: c.Input.AmountIn.Copy()})
			require.ErrorIs(t, err, ErrNotTracked, c.ID)
		}
		if strings.HasSuffix(c.ID, "gldHookedOnly") {
			require.Nil(t, h.Market, c.ID)
		}
	}
	t.Logf("%d fixtures tracked, %d published empty", len(cases), empty)
}
