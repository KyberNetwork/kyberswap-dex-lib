package flywheel

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestFeeAndInverseRounding(t *testing.T) {
	h := &Hook{}
	for n := int64(1); n <= 10000; n++ {
		for _, direction := range []bool{true, false} {
			input := big.NewInt(n)
			p := &v4.BeforeSwapParams{CalcOut: true, ZeroForOne: direction, AmountSpecified: input}
			before, err := h.BeforeSwap(p)
			require.NoError(t, err)
			require.Zero(t, before.DeltaSpecified.Sign())
			after, err := h.AfterSwap(&v4.AfterSwapParams{BeforeSwapParams: p, AmountOut: input})
			require.NoError(t, err)
			require.Equal(t, n/25, after.HookFee.Int64())
			require.Equal(t, n, input.Int64(), "fee calculation must not mutate the supplied amount")
			p.CalcOut = false
			before, err = h.BeforeSwap(p)
			require.NoError(t, err)
			gross := n + before.DeltaSpecified.Int64()
			require.Equal(t, n, gross-gross/25)
			require.Less(t, gross-1-(gross-1)/25, n, "inverse must choose the minimum gross output")
			after, err = h.AfterSwap(&v4.AfterSwapParams{BeforeSwapParams: p})
			require.NoError(t, err)
			require.Zero(t, after.HookFee.Sign(), "reverse exact-in must not add an input fee")
		}
	}
}

func TestBoundsAndChain(t *testing.T) {
	h := &Hook{}
	for _, amount := range []*big.Int{nil, big.NewInt(0), big.NewInt(-1), new(big.Int).Lsh(big.NewInt(1), 127)} {
		for _, calcOut := range []bool{false, true} {
			_, err := h.BeforeSwap(&v4.BeforeSwapParams{CalcOut: calcOut, AmountSpecified: amount})
			require.ErrorIs(t, err, ErrInvalidAmount)
		}
		_, err := h.AfterSwap(&v4.AfterSwapParams{BeforeSwapParams: &v4.BeforeSwapParams{CalcOut: true}, AmountOut: amount})
		require.ErrorIs(t, err, ErrInvalidAmount)
	}
	max := new(big.Int).Sub(new(big.Int).Lsh(big.NewInt(1), 127), big.NewInt(1))
	net := new(big.Int).Sub(max, new(big.Int).Quo(max, big.NewInt(25)))
	_, err := h.BeforeSwap(&v4.BeforeSwapParams{AmountSpecified: net})
	require.NoError(t, err)
	_, err = h.BeforeSwap(&v4.BeforeSwapParams{AmountSpecified: new(big.Int).Add(net, big.NewInt(1))})
	require.ErrorIs(t, err, ErrInvalidAmount)
	for _, chain := range []valueobject.ChainID{valueobject.ChainIDRobinhood, valueobject.ChainIDEthereum} {
		hook, ok := v4.GetHook(HookAddress, &v4.HookParam{Cfg: &v4.Config{ChainID: chain}})
		require.True(t, ok)
		_, err = hook.Track(context.Background(), nil) // No RPC client is needed.
		if chain != valueobject.ChainIDRobinhood {
			require.ErrorIs(t, err, ErrUnsupportedChain)
			_, err = hook.BeforeSwap(&v4.BeforeSwapParams{CalcOut: true, AmountSpecified: big.NewInt(100)})
			require.ErrorIs(t, err, ErrUnsupportedChain)
		} else {
			require.NoError(t, err)
			require.Equal(t, valueobject.ExchangeUniswapV4Flywheel, hook.GetExchange())
			require.Empty(t, hook.GetHookData())
			require.IsType(t, &Hook{}, hook.CloneState())
			require.NotSame(t, hook, hook.CloneState())
		}
	}
}

func testPool(t *testing.T, hook common.Address) *v4.PoolSimulator {
	t.Helper()
	static, err := json.Marshal(v4.StaticExtra{HooksAddress: hook, TickSpacing: 200})
	require.NoError(t, err)
	sim, err := v4.NewPoolSimulator(entity.Pool{
		Address: "0x1111111111111111111111111111111111111111111111111111111111111111",
		Type:    v4.DexType, Exchange: v4.DexType, BlockNumber: 87735162,
		Reserves: []string{"1000000000000", "1000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xe000000000000000000000000000000000000000", Decimals: 6, Swappable: true},
			{Address: "0xf000000000000000000000000000000000000000", Decimals: 18, Swappable: true},
		},
		Extra: `{"liquidity":1000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":200,"tick":0,` +
			`"ticks":[{"index":-887200,"liquidityGross":1000000000000,"liquidityNet":1000000000000},` +
			`{"index":887200,"liquidityGross":1000000000000,"liquidityNet":-1000000000000}]}`,
		StaticExtra: string(static),
	}, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	return sim
}

func TestSimulatorFeesReverseQuotesAndClones(t *testing.T) {
	for _, direction := range []int{0, 1} {
		sim, core := testPool(t, HookAddress), testPool(t, common.Address{})
		tokens := sim.GetTokens()
		args := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[direction], Amount: big.NewInt(1000000)}, TokenOut: tokens[1-direction]}
		gross, err := core.CalcAmountOut(args)
		require.NoError(t, err)
		net, err := sim.CalcAmountOut(args)
		require.NoError(t, err)
		want := new(big.Int).Sub(gross.TokenAmountOut.Amount, new(big.Int).Quo(gross.TokenAmountOut.Amount, big.NewInt(25)))
		require.Equal(t, want, net.TokenAmountOut.Amount)
		repeat, err := sim.CalcAmountOut(args)
		require.NoError(t, err)
		require.Equal(t, net, repeat)
		for _, desired := range []int64{1, 24, 25, 26, 960000, 10000000000} {
			r, err := sim.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: args.TokenOut, Amount: big.NewInt(desired)}, TokenIn: args.TokenAmountIn.Token})
			require.NoError(t, err)
			p := args
			p.TokenAmountIn.Amount = r.TokenAmountIn.Amount
			out, err := sim.CalcAmountOut(p)
			require.NoError(t, err)
			require.GreaterOrEqual(t, out.TokenAmountOut.Amount.Int64(), desired)
			p.TokenAmountIn.Amount = new(big.Int).Sub(r.TokenAmountIn.Amount, big.NewInt(1))
			less, err := sim.CalcAmountOut(p)
			if err == nil {
				require.Less(t, less.TokenAmountOut.Amount.Int64(), desired)
			}
		}
		clone := sim.CloneState()
		clone.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: args.TokenAmountIn, TokenAmountOut: *net.TokenAmountOut, Fee: *net.Fee, SwapInfo: net.SwapInfo})
		updated, err := clone.CalcAmountOut(args)
		require.NoError(t, err)
		require.Less(t, updated.TokenAmountOut.Amount.Cmp(net.TokenAmountOut.Amount), 0)
		original, err := sim.CalcAmountOut(args)
		require.NoError(t, err)
		require.Equal(t, net, original, "clone update must not mutate original")
		meta := sim.GetMetaInfo(args.TokenAmountIn.Token, args.TokenOut).(v4.PoolMetaInfo)
		require.Equal(t, HookAddress, meta.HookAddress)
		require.Equal(t, uint64(87735162), meta.BlockNumber)
		require.Empty(t, meta.HookData)
	}
}

func TestExhaustedLiquidityAndFailedQuotePurity(t *testing.T) {
	for _, direction := range []int{0, 1} {
		sim := testPool(t, HookAddress)
		tokens := sim.GetTokens()
		args := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: tokens[direction], Amount: big.NewInt(1000000)}, TokenOut: tokens[1-direction]}
		before, err := sim.CalcAmountOut(args)
		require.NoError(t, err)
		bad := args
		bad.TokenAmountIn.Amount = new(big.Int).Lsh(big.NewInt(1), 128)
		_, err = sim.CalcAmountOut(bad)
		require.Error(t, err)
		large := args
		large.TokenAmountIn.Amount = new(big.Int).Exp(big.NewInt(10), big.NewInt(35), nil)
		_, err = sim.CalcAmountOut(large)
		require.Error(t, err)
		// Preserve the standard V4 engine's rejection beyond the tracked ticks.
		// Do not fabricate a fill or consume input after liquidity is exhausted.
		_, coreErr := testPool(t, common.Address{}).CalcAmountOut(large)
		require.Equal(t, coreErr, err)
		after, err := sim.CalcAmountOut(args)
		require.NoError(t, err)
		require.Equal(t, before, after)
	}
}
