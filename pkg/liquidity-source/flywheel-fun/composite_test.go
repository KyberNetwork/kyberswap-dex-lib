package flywheelfun

import (
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// These are real replacement-contract snapshots. External bases were captured
// at the same block; router-service supplies those independently indexed pools.
func snapshots(t testing.TB) []entity.Pool {
	t.Helper()
	b, e := os.ReadFile("testdata/replacement-fork-markets.json")
	require.NoError(t, e)
	var captured []entity.Pool
	require.NoError(t, json.Unmarshal(b, &captured))
	require.Len(t, captured, 9)
	out := make([]entity.Pool, 0, len(captured))
	for _, i := range []int{0, 1, 3, 2, 4, 5, 6, 7, 8} {
		out = append(out, withIndexedBases(t, captured[i]))
	}
	return out
}
func withIndexedBases(t testing.TB, p entity.Pool) entity.Pool {
	t.Helper()
	var st StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &st))
	var ex struct {
		RoutePools []entity.Pool `json:"routePools"`
	}
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &ex))
	st.BasePools = nil
	for _, ep := range ex.RoutePools {
		st.BasePools = append(st.BasePools, ep.Address)
	}
	b, e := json.Marshal(st)
	require.NoError(t, e)
	p.StaticExtra = string(b)
	return p
}

func baseSim(t testing.TB, ep entity.Pool) pool.IPoolSimulator {
	t.Helper()
	var s pool.IPoolSimulator
	var err error
	if ep.Type == v3.DexTypeUniswapV3 {
		s, err = v3.NewPoolSimulator(ep, 4663)
	} else {
		s, err = v4.NewPoolSimulator(ep, 4663)
	}
	require.NoError(t, err)
	return s
}

// baseMap stands in for router-service's basePoolMap of indexed uniswap pools.
func baseMap(t testing.TB, p entity.Pool) map[string]pool.IPoolSimulator {
	t.Helper()
	var ex struct {
		RoutePools []entity.Pool `json:"routePools"`
	}
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &ex))
	out := make(map[string]pool.IPoolSimulator, len(ex.RoutePools))
	for _, ep := range ex.RoutePools {
		require.Equal(t, p.BlockNumber, ep.BlockNumber)
		out[ep.Address] = baseSim(t, ep)
	}
	return out
}

func newSim(t testing.TB, p entity.Pool) *PoolSimulator {
	t.Helper()
	s, err := NewPoolSimulatorWithBases(p, baseMap(t, p))
	require.NoError(t, err)
	return s
}
func quoteBuy(t *testing.T, s *PoolSimulator, amount int64) *pool.CalcAmountOutResult {
	t.Helper()
	r, e := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: big.NewInt(amount)}, TokenOut: s.Info.Address})
	require.NoError(t, e)
	return r
}

// Route hops must run on the indexed uniswap pools (no duplicate source), and a
// flywheel clone must own its bases so pathfinder state never leaks into originals.
func TestBasePoolsFromBaseMap(t *testing.T) {
	p := snapshots(t)[2]
	_, err := NewPoolSimulator(p)
	require.ErrorIs(t, err, ErrBasePool, "a routed launch is unquotable without its indexed base pool")
	bases := baseMap(t, p)
	s, err := NewPoolSimulatorWithBases(p, bases)
	require.NoError(t, err)
	base := s.GetBasePools()[0]
	require.Same(t, bases[base.GetAddress()], base, "uses the indexed pool's simulator, not a snapshot")
	price := fingerprint(base)
	q := quoteBuy(t, s, 1000000000)
	require.Equal(t, price, fingerprint(base), "quotes do not mutate the base")

	clone := s.CloneState().(*PoolSimulator)
	clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: q.SwapInfo})
	require.True(t, clone.Valid)
	require.Equal(t, price, fingerprint(base), "clones own their bases")
	direct := func(b pool.IPoolSimulator) *big.Int {
		r, e := b.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: big.NewInt(1000000000)}, TokenOut: s.Static.Quote})
		require.NoError(t, e)
		return r.TokenAmountOut.Amount
	}
	require.Negative(t, direct(clone.Base).Cmp(direct(base)), "the route leg consumed the clone's base liquidity")
	require.Negative(t, quoteBuy(t, clone, 1000000000).TokenAmountOut.Amount.Cmp(q.TokenAmountOut.Amount))

	clone.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: q.SwapInfo})
	require.False(t, clone.Valid, "a stale quote is rejected")
	relinked := s.CloneState().(*PoolSimulator)
	relinked.SetBasePool(base)
	require.Same(t, base, relinked.Base)
	relinked.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: q.SwapInfo})
	require.False(t, relinked.Valid, "quotes taken on the old base no longer apply")
}
func TestCompositeSnapshotsPurityAndProtection(t *testing.T) {
	for _, p := range snapshots(t) {
		s := newSim(t, p)
		first := quoteBuy(t, s, 1000000000)
		second := quoteBuy(t, s, 1000000000)
		require.Equal(t, first, second)
		clone := s.CloneState().(*PoolSimulator)
		s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: first.SwapInfo})
		require.Equal(t, second.TokenAmountOut.Amount, quoteBuy(t, clone, 1000000000).TokenAmountOut.Amount)
		large, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: new(big.Int).Lsh(big.NewInt(1), 255)}, TokenOut: s.Info.Address})
		if s.Static.Quote == WETH && !s.Curve.Graduated {
			require.NoError(t, err)
			require.Positive(t, large.RemainingTokenAmountIn.Amount.Sign())
			require.True(t, large.SwapInfo.(SwapInfo).Next.Graduated)
		} else {
			require.Error(t, err)
		}
	}
	for _, a := range []string{Factory, Settlement, MarketAdapter, WETH, Manager, NativeHook} {
		require.True(t, common.IsHexAddress(a), a)
	}
}
func TestDirectionalProtocolFeeAndCalldata(t *testing.T) {
	p := snapshots(t)[0]
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	e.Protocol = [2]uint32{1000, 1}
	raw, _ := json.Marshal(e)
	p.Extra = string(raw)
	s, err := NewPoolSimulator(p)
	require.NoError(t, err)
	q := quoteBuy(t, s, 10000000000)
	info := q.SwapInfo.(SwapInfo)
	rate := e.Protocol[s.MarketPool.GetTokenIndex(WETH)]
	platform, err := BuyPlatform(uint256.NewInt(10000000000), rate)
	require.NoError(t, err)
	gross := new(uint256.Int).Sub(uint256.NewInt(10000000000), &platform)
	split, err := BuyQuote(gross, rate)
	require.NoError(t, err)
	isolated := s.MarketPool.CloneState().(*v3.PoolSimulator)
	out, _, _, err := swapCL(isolated, &split.Net, WETH, s.Info.Address, rate, -1)
	require.NoError(t, err)
	require.Equal(t, out.ToBig(), q.TokenAmountOut.Amount)
	data, err := EncodeTradeData(info, 75, 100, 400)
	require.NoError(t, err)
	decoded, err := tradeArguments.Unpack(data)
	require.NoError(t, err)
	trade := abi.ConvertType(decoded[0], new(AdapterTrade)).(*AdapterTrade)
	require.Equal(t, new(big.Int).Div(new(big.Int).Mul(gross.ToBig(), big.NewInt(9925)), big.NewInt(10000)), trade.MinQuote)
	require.Zero(t, trade.MinRefundETH.Sign())
	require.Empty(t, trade.Route)
	s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: info})
	sell, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *q.TokenAmountOut, TokenOut: WETH})
	require.NoError(t, err)
	require.False(t, sell.SwapInfo.(SwapInfo).Buy)
	e.Protocol[0] = 1001
	raw, _ = json.Marshal(e)
	p.Extra = string(raw)
	_, err = NewPoolSimulator(p)
	require.Error(t, err)
}

// syntheticBase is a full-range WETH/quote base pool: kind 3 = uniswapv3, 4 = native-ETH uniswap-v4.
func syntheticBase(t *testing.T, quote string, kind uint8, block uint64, forgedID string) (pool.IPoolSimulator, string) {
	t.Helper()
	liq := new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)
	key := poolKey{Currency1: common.HexToAddress(quote), Fee: big.NewInt(3000), TickSpacing: big.NewInt(60)}
	id, err := keyID(key)
	require.NoError(t, err)
	extra := v3.Extra{Liquidity: liq, SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96), Tick: big.NewInt(0), TickSpacing: 60, Ticks: []v3.Tick{{Index: -60000, LiquidityGross: liq, LiquidityNet: liq}, {Index: 60000, LiquidityGross: liq, LiquidityNet: new(big.Int).Neg(liq)}}}
	if forgedID != "" {
		id = forgedID
	}
	ep, err := poolEntity(id, key, new(big.Int).SetUint64(block), extra, 3000)
	require.NoError(t, err)
	if kind == 3 {
		ep.Address, ep.Type, ep.Exchange, ep.StaticExtra = "0x1234567890123456789012345678901234567890", v3.DexTypeUniswapV3, v3.DexTypeUniswapV3, ""
	}
	return baseSim(t, ep), ep.Address
}

// The settlement route is rebuilt from the indexed base pool itself, so config needs only its id.
func TestRouteFromBasePool(t *testing.T) {
	p := snapshots(t)[2]
	var st StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &st))
	for _, kind := range []uint8{3, 4} {
		base, id := syntheticBase(t, st.Quote, kind, p.BlockNumber, "")
		st.BasePools = []string{id}
		raw, _ := json.Marshal(st)
		p.StaticExtra = string(raw)
		s, err := NewPoolSimulatorWithBases(p, map[string]pool.IPoolSimulator{id: base})
		require.NoError(t, err)
		q := quoteBuy(t, s, 10000000000)
		info := q.SwapInfo.(SwapInfo)
		require.Equal(t, []byte{0x4e, 0x41, 0x54, 0x31}, []byte(info.Route[:4]))
		decoded, err := routeArguments.Unpack(info.Route[4:])
		require.NoError(t, err)
		hops := *abi.ConvertType(decoded[0], new([]routeHop)).(*[]routeHop)
		require.Len(t, hops, 1)
		require.Equal(t, kind, hops[0].Kind)
		if kind == 3 {
			require.Equal(t, common.HexToAddress(id), hops[0].Pool)
		} else {
			require.Equal(t, common.Address{}, hops[0].Pool)
			require.Equal(t, common.Address{}, hops[0].Key.Currency0, "native ETH side")
			require.Equal(t, common.HexToAddress(st.Quote), hops[0].Key.Currency1)
			require.Equal(t, int64(3000), hops[0].Key.Fee.Int64())
			require.Equal(t, int64(60), hops[0].Key.TickSpacing.Int64())
		}
		s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: info})
		sell, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *q.TokenAmountOut, TokenOut: WETH})
		require.NoError(t, err)
		require.Equal(t, info.Route, sell.SwapInfo.(SwapInfo).Route, "one hop encodes the same both ways")
		require.Less(t, sell.TokenAmountOut.Amount.Cmp(big.NewInt(10000000000)), 0)
	}
	base, id := syntheticBase(t, st.Quote, 4, p.BlockNumber, "")
	other, _ := syntheticBase(t, "0x3000000000000000000000000000000000000000", 4, p.BlockNumber, "")
	forged, forgedID := syntheticBase(t, st.Quote, 4, p.BlockNumber, "0x"+strings.Repeat("ab", 32))
	for name, c := range map[string]struct {
		basePools []string
		bases     map[string]pool.IPoolSimulator
		err       error
	}{
		"missing base":     {[]string{id}, nil, ErrBasePool},
		"two hops":         {[]string{id, id}, map[string]pool.IPoolSimulator{id: base}, ErrState},
		"base lacks quote": {[]string{id}, map[string]pool.IPoolSimulator{id: other}, ErrState},
		"id not the key":   {[]string{forgedID}, map[string]pool.IPoolSimulator{forgedID: forged}, ErrState},
	} {
		st.BasePools = c.basePools
		raw, _ := json.Marshal(st)
		p.StaticExtra = string(raw)
		_, err := NewPoolSimulatorWithBases(p, c.bases)
		require.ErrorIs(t, err, c.err, name)
	}
}

// The settlement reverts buyWithRefund on a full fill and buy on a partial one, so the
// encoder reads SwapInfo.Refunds (via the route-summary JSON) to pick the entry point.
func TestRefundsFlagReachesEncoder(t *testing.T) {
	for _, p := range snapshots(t) {
		s := newSim(t, p)
		if s.Static.Quote != WETH || s.Curve.Graduated {
			continue
		}
		small := quoteBuy(t, s, 1000000000).SwapInfo.(SwapInfo)
		require.False(t, small.Refunds)
		large, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: new(big.Int).Lsh(big.NewInt(1), 100)}, TokenOut: s.Info.Address})
		if err != nil {
			continue
		}
		raw, err := json.Marshal(large.SwapInfo)
		require.NoError(t, err)
		var m map[string]any
		require.NoError(t, json.Unmarshal(raw, &m))
		require.Equal(t, true, m["refunds"])
		return
	}
	t.Skip("no curve snapshot crosses graduation")
}
