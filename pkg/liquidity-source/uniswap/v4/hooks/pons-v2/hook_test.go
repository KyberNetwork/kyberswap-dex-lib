package ponsv2

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// AfterSwap must mirror PonsV2MemeHook._afterSwap: feeAmount + taxAmount = floor(unspecified
// * (hookFeeBps + creatorTaxBps) / BASIS_POINTS), taken from the realized output on exact-in.
func TestAfterSwap_ExactIn_ChargesFeeOnOutput(t *testing.T) {
	h := &Hook{Registered: true, FeeBps: 100 + 50} // hookFeeBps=100, creatorTaxBps=50

	result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountOut:        big.NewInt(1_000_000),
	})
	assert.NoError(t, err)
	// floor(1_000_000 * 150 / 10_000) = 15_000
	assert.Equal(t, big.NewInt(15_000), result.HookFee)
}

// CalcAmountIn reverses an exact-in swap, and the hook takes its fee from the realized
// output, so AfterSwap must not add a fee to the input (BeforeSwap grosses the output up).
func TestAfterSwap_CalcIn_NoInputFee(t *testing.T) {
	h := &Hook{Registered: true, FeeBps: 150}

	result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: false},
		AmountIn:         big.NewInt(1_000_000),
	})
	assert.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, result.HookFee)
}

// The pool must output `net` + fee so that the output fee leaves `net`: the smallest gross
// with gross - floor(gross * 150 / 10_000) >= net.
func TestBeforeSwap_CalcIn_GrossesOutputUp(t *testing.T) {
	h := &Hook{Registered: true, FeeBps: 150}

	result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, AmountSpecified: big.NewInt(985_000)})
	require.NoError(t, err)
	// gross = floor(984_999 * 10_000 / 9_850) + 1 = 999_999; 999_999 - floor(14_999.985) = 985_000
	assert.Equal(t, big.NewInt(14_999), result.DeltaSpecified)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaUnspecified)

	result, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: big.NewInt(985_000)})
	require.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, result.DeltaSpecified, "CalcOut: the fee comes from AfterSwap")
}

func TestBeforeSwap_NoFee_NoOp(t *testing.T) {
	for _, h := range []*Hook{{}, {Registered: true}} {
		result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, AmountSpecified: big.NewInt(985_000)})
		require.NoError(t, err)
		assert.Equal(t, bignumber.ZeroBI, result.DeltaSpecified)
	}
}

// Executing the quoted input exact-in must deliver at least the requested output, and one
// wei less input must not. It failed when the fee was charged on the input in CalcAmountIn.
func TestCalcAmountIn_DeliversRequestedOutput(t *testing.T) {
	sim := newTestSim(t, HookAddresses[0], `{"r":true,"f":233}`)
	tokens := sim.Info.Tokens
	want := big.NewInt(1_000_000)
	delivered := func(in *big.Int) *big.Int {
		res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
			TokenAmountIn: pool.TokenAmount{Token: tokens[0], Amount: in}, TokenOut: tokens[1]})
		require.NoError(t, err)
		return res.TokenAmountOut.Amount
	}

	res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
		TokenAmountOut: pool.TokenAmount{Token: tokens[1], Amount: want}, TokenIn: tokens[0]})
	require.NoError(t, err)
	in := res.TokenAmountIn.Amount
	require.GreaterOrEqual(t, delivered(in).Cmp(want), 0, "quoted input must deliver the requested output")
	require.Less(t, delivered(new(big.Int).Sub(in, bignumber.One)).Cmp(want), 0, "and be the smallest such input")
}

// newTestSim builds a v4 pool simulator for a 1:1 full-range pool behind the hook at addr.
func newTestSim(t *testing.T, addr common.Address, hookExtra string) *uniswapv4.PoolSimulator {
	t.Helper()
	static, err := json.Marshal(map[string]any{"0x0": [2]bool{false, false}, "fee": 2000, "tS": 60, "hooks": addr})
	require.NoError(t, err)
	extra := `{"liquidity":1000000000000,"sqrtPriceX96":79228162514264337593543950336,"tickSpacing":60,"tick":0,` +
		`"ticks":[{"index":-887220,"liquidityGross":1000000000000,"liquidityNet":1000000000000},` +
		`{"index":887220,"liquidityGross":1000000000000,"liquidityNet":-1000000000000}],"hX":` + hookExtra + `}`
	p, err := uniswapv4.NewPoolSimulator(entity.Pool{
		Address:  "0x1111111111111111111111111111111111111111111111111111111111111111",
		Type:     uniswapv4.DexType,
		SwapFee:  2000,
		Reserves: []string{"1000000000000", "1000000000000"},
		Tokens: []*entity.PoolToken{
			{Address: "0xe000000000000000000000000000000000000000", Decimals: 18, Swappable: true},
			{Address: "0xf000000000000000000000000000000000000000", Decimals: 18, Swappable: true},
		},
		Extra: extra, StaticExtra: string(static),
	}, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	return p
}

// A pool never seen by Track() (RPC failure, or genuinely not yet registered with the
// factory) must price as fee-free rather than error -- unlike b20's LaunchHook, an
// unregistered PonsV2MemeHook pool really does charge nothing (_afterSwap's own
// `if (!info.registered) return 0` early-out), so refusing to quote would be wrong here.
func TestAfterSwap_Unregistered_NoFee(t *testing.T) {
	h := &Hook{}

	result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountOut:        big.NewInt(1_000_000),
	})
	assert.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, result.HookFee)
}

func TestAfterSwap_ZeroFee_NoOp(t *testing.T) {
	h := &Hook{Registered: true, FeeBps: 0}

	result, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountOut:        big.NewInt(1_000_000),
	})
	assert.NoError(t, err)
	assert.Equal(t, bignumber.ZeroBI, result.HookFee)
}
