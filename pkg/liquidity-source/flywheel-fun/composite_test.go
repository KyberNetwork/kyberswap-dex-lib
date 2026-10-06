package flywheelfun

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func snapshots(t *testing.T) []entity.Pool {
	t.Helper()
	b, e := os.ReadFile("testdata/fork-markets.json")
	require.NoError(t, e)
	var out []entity.Pool
	require.NoError(t, json.Unmarshal(b, &out))
	require.GreaterOrEqual(t, len(out), 5)
	return out
}

func TestRouteCacheUsesBlockHashAndBounds(t *testing.T) {
	p := snapshots(t)[2]
	var e Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	h := e.Route[0]
	id, err := h.id()
	require.NoError(t, err)
	client, m := fixtureRPC(t)
	tracker, err := NewPoolTracker(&Config{}, client)
	require.NoError(t, err)
	hash := m.header.Hash()
	tracker.routeBlock = hash
	tracker.routeCache = map[string]routeSnapshot{id: {e.RoutePools[0], h}}
	_, _, err = tracker.cachedRoute(context.Background(), h, m.header.Number, hash)
	require.NoError(t, err)
	require.Zero(t, m.calls)
	m.fail = true
	m.header.Extra = []byte("reorg")
	_, _, err = tracker.cachedRoute(context.Background(), h, m.header.Number, m.header.Hash())
	require.Error(t, err)
	require.Equal(t, 1, m.calls)
	require.Empty(t, tracker.routeCache)
	_, _, err = wordRange(1)
	require.ErrorIs(t, err, ErrUnsupported)
	_, _, err = wordRange(0)
	require.ErrorIs(t, err, ErrState)
	lo, hi, err := wordRange(60)
	require.NoError(t, err)
	require.LessOrEqual(t, hi-lo+1, 256)
}
func quoteBuy(t *testing.T, s *PoolSimulator, amount int64) *pool.CalcAmountOutResult {
	t.Helper()
	r, e := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: WETH, Amount: big.NewInt(amount)}, TokenOut: s.Info.Address})
	require.NoError(t, e)
	return r
}
func TestSharedRouteLiquidityAndStaleQuote(t *testing.T) {
	p := snapshots(t)[2]
	a, e := NewPoolSimulator(p)
	require.NoError(t, e)
	require.Len(t, a.GetBasePools(), 1)
	base := a.GetBasePools()[0]
	p.Address = "0x2222222222222222222222222222222222222222"
	p.Tokens = entity.ClonePoolTokens(p.Tokens)
	p.Tokens[1].Address = p.Address
	b, e := NewPoolSimulatorWithBases(p, map[string]pool.IPoolSimulator{base.GetAddress(): base})
	require.NoError(t, e)
	before := quoteBuy(t, b, 1000000000000)
	clone := b.CloneState().(*PoolSimulator)
	qa := quoteBuy(t, a, 1000000000000)
	a.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: qa.SwapInfo})
	after := quoteBuy(t, b, 1000000000000)
	require.Less(t, after.TokenAmountOut.Amount.Cmp(before.TokenAmountOut.Amount), 0, "second candidate must see the first route's price impact")
	require.Equal(t, before.TokenAmountOut.Amount, quoteBuy(t, clone, 1000000000000).TokenAmountOut.Amount)
	current := fingerprint(base)
	b.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: before.SwapInfo})
	require.False(t, b.Valid)
	require.Equal(t, current, fingerprint(base), "stale update cannot partially mutate shared liquidity")
}
func TestCompositeSnapshotsPurityAndProtection(t *testing.T) {
	for _, p := range snapshots(t) {
		s, e := NewPoolSimulator(p)
		require.NoError(t, e)
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
	for _, a := range []string{Factory, Settlement, MarketAdapter, WETH, Manager, Market, NativeHook, V3Factory} {
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
	isolated := s.MarketPool.CloneState()
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
func syntheticHop(t *testing.T, a, b string, kind uint8, block uint64) (RouteHop, entity.Pool) {
	t.Helper()
	liq := new(big.Int).Exp(big.NewInt(10), big.NewInt(24), nil)
	key := PoolKey{Currency0: common.HexToAddress(a), Currency1: common.HexToAddress(b), Fee: big.NewInt(3000), TickSpacing: big.NewInt(60)}
	if key.Currency0.Big().Cmp(key.Currency1.Big()) > 0 {
		key.Currency0, key.Currency1 = key.Currency1, key.Currency0
	}
	h := RouteHop{Kind: kind, Key: key, LPFee: 3000}
	if kind == 3 {
		h.Pool = common.HexToAddress("0x1234567890123456789012345678901234567890")
	}
	id, err := h.id()
	require.NoError(t, err)
	extra := v3.Extra{Liquidity: liq, SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96), Tick: big.NewInt(0), TickSpacing: 60, Ticks: []v3.Tick{{Index: -60000, LiquidityGross: liq, LiquidityNet: liq}, {Index: 60000, LiquidityGross: liq, LiquidityNet: new(big.Int).Neg(liq)}}}
	ep, err := poolEntity(id, key, new(big.Int).SetUint64(block), extra, 3000, kind)
	require.NoError(t, err)
	return h, ep
}
func TestMixedRouteOrderingAndValidation(t *testing.T) {
	p := snapshots(t)[2]
	var e Extra
	var st StaticExtra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &e))
	require.NoError(t, json.Unmarshal([]byte(p.StaticExtra), &st))
	mid := "0x3000000000000000000000000000000000000000"
	a, ap := syntheticHop(t, WETH, mid, 3, p.BlockNumber)
	b, bp := syntheticHop(t, mid, st.Quote, 4, p.BlockNumber)
	e.Route = []RouteHop{a, b}
	e.RoutePools = []entity.Pool{ap, bp}
	raw, _ := json.Marshal(e)
	p.Extra = string(raw)
	s, err := NewPoolSimulator(p)
	require.NoError(t, err)
	q := quoteBuy(t, s, 10000000000)
	info := q.SwapInfo.(SwapInfo)
	require.Len(t, info.Steps, 2)
	data, err := EncodeRoute(info.Route, true)
	require.NoError(t, err)
	require.Equal(t, []byte{0x4e, 0x41, 0x54, 0x31}, data[:4])
	decoded, err := routeArguments.Unpack(data[4:])
	require.NoError(t, err)
	hops := abi.ConvertType(decoded[0], new([]encodedHop)).(*[]encodedHop)
	require.Equal(t, uint8(4), (*hops)[0].Kind)
	require.Equal(t, uint8(3), (*hops)[1].Kind)
	s.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: info})
	sell, err := s.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: *q.TokenAmountOut, TokenOut: WETH})
	require.NoError(t, err)
	require.Less(t, sell.TokenAmountOut.Amount.Cmp(big.NewInt(10000000000)), 0)
	// Wrong direction, duplicate/cyclic legs, unsupported hooks and mismatched blocks.
	for _, mutate := range []func(*Extra){func(e *Extra) { e.Route[0], e.Route[1] = e.Route[1], e.Route[0] }, func(e *Extra) {
		e.Route = append(e.Route, e.Route[0])
		e.RoutePools = append(e.RoutePools, e.RoutePools[0])
	}, func(e *Extra) { e.Route[1].Key.Hooks = common.HexToAddress(NativeHook) }, func(e *Extra) { e.RoutePools[0].BlockNumber++ }} {
		var broken Extra
		require.NoError(t, json.Unmarshal(raw, &broken))
		mutate(&broken)
		encoded, _ := json.Marshal(broken)
		bad := p
		bad.Extra = string(encoded)
		_, err = NewPoolSimulator(bad)
		require.Error(t, err)
	}
}
