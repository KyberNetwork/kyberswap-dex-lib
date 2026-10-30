package reafund

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	ponsv2 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/pons-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The hook address resolves to this package under REA.fund's exchange, with Extra restored from
// HookExtra, and carries Pons' hook's flags (beforeInitialize, afterSwap, afterSwapReturnDelta).
func TestHookFactory(t *testing.T) {
	addr := HookAddresses[0]
	hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(`{"r":true,"f":200}`)})
	require.True(t, ok)
	h, ok := hook.(*Hook)
	require.True(t, ok)
	assert.Equal(t, valueobject.ExchangeUniswapV4Reafund, h.GetExchange())
	assert.Equal(t, ponsv2.Extra{Registered: true, FeeBps: 200}, h.Extra)

	flags := func(a common.Address) uint16 { return (uint16(a[18])<<8 | uint16(a[19])) & 0x3fff }
	assert.Equal(t, flags(ponsv2.HookAddresses[0]), flags(addr))
	assert.True(t, h.CanAfterSwap(addr))
	assert.True(t, h.CanBeforeSwap(addr), "simulator-only, for CalcAmountIn's gross-up")
}

// A pool behind REA.fund's hook quotes what the same pool behind Pons' hook quotes, both ways and
// in both directions, and an exact-out quote's input is the smallest that delivers the output.
func TestQuotesMatchPonsV2(t *testing.T) {
	ours, pons := newTestSim(t, HookAddresses[0]), newTestSim(t, ponsv2.HookAddresses[0])
	tokens := ours.Info.Tokens
	swapOut := func(sim *uniswapv4.PoolSimulator, in, out int, amount *big.Int) *big.Int {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokens[in], Amount: amount}, TokenOut: tokens[out]})
		require.NoError(t, err)
		return res.TokenAmountOut.Amount
	}
	swapIn := func(sim *uniswapv4.PoolSimulator, in, out int, amount *big.Int) *big.Int {
		res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: tokens[out], Amount: amount}, TokenIn: tokens[in]})
		require.NoError(t, err)
		return res.TokenAmountIn.Amount
	}

	for _, dir := range [][2]int{{0, 1}, {1, 0}} {
		in, out := dir[0], dir[1]
		for _, amount := range []*big.Int{big.NewInt(1_000), big.NewInt(1_000_000), big.NewInt(10_000_000_000)} {
			assert.Equal(t, swapOut(pons, in, out, amount), swapOut(ours, in, out, amount), "exact-in %s", amount)

			quoted := swapIn(ours, in, out, amount)
			assert.Equal(t, swapIn(pons, in, out, amount), quoted, "exact-out %s", amount)
			assert.GreaterOrEqual(t, swapOut(ours, in, out, quoted).Cmp(amount), 0, "quoted input delivers %s", amount)
			assert.Negative(t, swapOut(ours, in, out, new(big.Int).Sub(quoted, bignumber.One)).Cmp(amount),
				"one wei less does not deliver %s", amount)
		}
	}
}

// What Track returns is what the factory restores from HookExtra on the next load.
func TestTrackOutputRestores(t *testing.T) {
	h := &Hook{Extra: ponsv2.Extra{Registered: true, FeeBps: 100}}
	raw, err := json.Marshal(h)
	require.NoError(t, err)
	assert.JSONEq(t, `{"r":true,"f":100}`, string(raw))

	restored, ok := uniswapv4.GetHook(HookAddresses[0], &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(raw)})
	require.True(t, ok)
	assert.Equal(t, h.Extra, restored.(*Hook).Extra)
}

// newTestSim is a 1:1 full-range pool behind the hook at addr, registered with a 2.33% fee.
func newTestSim(t *testing.T, addr common.Address) *uniswapv4.PoolSimulator {
	t.Helper()
	static, err := json.Marshal(map[string]any{"0x0": [2]bool{false, false}, "fee": 2000, "tS": 60, "hooks": addr})
	require.NoError(t, err)
	extra := `{"liquidity":1000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,` +
		`"ticks":[{"index":-887220,"liquidityGross":1000000000000,"liquidityNet":1000000000000},` +
		`{"index":887220,"liquidityGross":1000000000000,"liquidityNet":-1000000000000}],"hX":{"r":true,"f":233}}`
	sim, err := uniswapv4.NewPoolSimulator(entity.Pool{
		Address:  "0x1111111111111111111111111111111111111111111111111111111111111111",
		Type:     uniswapv4.DexType,
		SwapFee:  2000,
		Reserves: []string{"1000000000000", "1000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xe000000000000000000000000000000000000000", Decimals: 18, Swappable: true},
			{Address: "0xf000000000000000000000000000000000000000", Decimals: 18, Swappable: true},
		},
		Extra: extra, StaticExtra: string(static),
	}, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	return sim
}
