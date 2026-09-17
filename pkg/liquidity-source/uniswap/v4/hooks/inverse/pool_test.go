package inverse

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestV4DispatchAndSequentialUpdate(t *testing.T) {
	addr := common.HexToAddress("0x0000000000000000000000000000000000002aec")
	old, exists := uniswapv4.HookFactories[addr]
	uniswapv4.RegisterHooksFactory(NewHook, addr)
	t.Cleanup(func() {
		if exists {
			uniswapv4.HookFactories[addr] = old
		} else {
			delete(uniswapv4.HookFactories, addr)
		}
	})
	vs := vectors(t)
	s := vs[0].Before
	raw, err := json.Marshal(&s)
	require.NoError(t, err)
	extra := uniswapv4.Extra{Extra: &uniswapv3.Extra{Liquidity: s.Liquidity.ToBig(), SqrtPriceX96: s.SqrtPriceX96.ToBig(), TickSpacing: 60, Tick: n(int64(s.Tick)), Ticks: []uniswapv3.Tick{{Index: minTick, LiquidityGross: s.Liquidity.ToBig(), LiquidityNet: s.Liquidity.ToBig()}, {Index: maxTick, LiquidityGross: s.Liquidity.ToBig(), LiquidityNet: new(big.Int).Neg(s.Liquidity.ToBig())}}}, HookExtra: raw}
	eb, err := json.Marshal(extra)
	require.NoError(t, err)
	se, err := json.Marshal(uniswapv4.StaticExtra{Fee: 3000, TickSpacing: 60, HooksAddress: addr})
	require.NoError(t, err)
	h := newHook(s)
	res, err := h.GetReserves(context.Background(), nil)
	require.NoError(t, err)
	p := entity.Pool{Address: "0x1234", Type: "uniswap-v4", Exchange: "uniswap-v4-inverse", BlockNumber: s.BlockNumber, SwapFee: 3000, Tokens: []*entity.PoolToken{{Address: "0x0000000000000000000000000000000000000001", Decimals: 18, Swappable: true}, {Address: "0x0000000000000000000000000000000000000002", Decimals: 18, Swappable: true}}, Reserves: res, Extra: string(eb), StaticExtra: string(se)}
	sim, err := uniswapv4.NewPoolSimulator(p, 4663)
	require.NoError(t, err)
	original := sim.CloneState().(*uniswapv4.PoolSimulator)
	// Fixture ordering is lexical; follow by state, not array index.
	for i := 0; i < 18; i++ {
		var v *vector
		for j := range vs {
			if vs[j].Before == s && vs[j].Success {
				v = &vs[j]
				break
			}
		}
		require.NotNil(t, v)
		in, out := p.Tokens[0].Address, p.Tokens[1].Address
		if !v.ZeroForOne {
			in, out = out, in
		}
		amount := pool.TokenAmount{Token: in, Amount: v.Input.ToBig()}
		result, e := sim.CalcAmountOut(pool.CalcAmountOutParams{TokenAmountIn: amount, TokenOut: out})
		require.NoError(t, e)
		require.Equal(t, v.Output.Dec(), result.TokenAmountOut.Amount.String())
		require.Empty(t, sim.GetMetaInfo(in, out).(uniswapv4.PoolMetaInfo).HookData)
		require.Equal(t, s.BlockNumber, sim.GetMetaInfo(in, out).(uniswapv4.PoolMetaInfo).BlockNumber)
		sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: amount, TokenAmountOut: *result.TokenAmountOut, Fee: *result.Fee, SwapInfo: result.SwapInfo})
		s = v.After
		expected := newHook(s).PoolState()
		require.Equal(t, expected.SqrtPriceX96, sim.V3Pool.SqrtRatioX96)
		require.Equal(t, expected.Liquidity, sim.V3Pool.Liquidity)
		for j := range expected.Reserves {
			require.Zero(t, expected.Reserves[j].Cmp(sim.Info.Reserves[j]))
		}
	}
	require.Equal(t, vs[0].Before.SqrtPriceX96, original.V3Pool.SqrtRatioX96)
	_, err = sim.CalcAmountIn(pool.CalcAmountInParams{TokenIn: p.Tokens[0].Address, TokenAmountOut: pool.TokenAmount{Token: p.Tokens[1].Address, Amount: n(1)}})
	require.ErrorIs(t, err, ErrExactOutput)
}
