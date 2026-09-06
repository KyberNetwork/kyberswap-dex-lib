package cashcat

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
)

func feeHook() *Hook {
	return &Hook{
		Hook:    &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Cashcat},
		FeeRate: big.NewInt(30000), // 3.0%: pool 0x6f0e... on Robinhood (CashCatHookV2.currentFeeRate)
	}
}

// TestFactory_ExchangeAndRestore covers registration: the hook resolves by
// address (taking precedence over the auto-detection fallback), reports its
// own exchange, and restores the tracked fee rate from HookExtra.
func TestFactory_ExchangeAndRestore(t *testing.T) {
	t.Parallel()

	hook, ok := uniswapv4.GetHook(common.HexToAddress("0x75a54357d9c78a2db19004a5fdc76c50f9242aec"), nil)
	require.True(t, ok)
	require.Equal(t, string(valueobject.ExchangeUniswapV4Cashcat), hook.GetExchange())

	restored, ok := uniswapv4.GetHook(
		common.HexToAddress("0x75a54357d9c78a2db19004a5fdc76c50f9242aec"),
		&uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(`{"f":30000}`)},
	)
	require.True(t, ok)
	cashcatHook, ok := restored.(*Hook)
	require.True(t, ok)
	require.Equal(t, "30000", cashcatHook.FeeRate.String())
}

// TestQuoteSpecified mirrors the hook's ethSpecified truth table.
func TestQuoteSpecified(t *testing.T) {
	t.Parallel()

	// CalcOut = exact input: quote specified only when selling currency0.
	require.True(t, quoteSpecified(true, true))
	require.False(t, quoteSpecified(true, false))
	// CalcIn = exact output: quote specified only when buying currency0.
	require.True(t, quoteSpecified(false, false))
	require.False(t, quoteSpecified(false, true))
}

// TestCalcOut_SellBaseTakesOutputFee mirrors the Tenderly ground truth for
// selling 264665 PURRS: beforeSwap delta (0,0), afterSwap takes exactly 3%
// of the quote moved.
func TestCalcOut_SellBaseTakesOutputFee(t *testing.T) {
	t.Parallel()
	h := feeHook()

	before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut: true, ZeroForOne: false,
		AmountSpecified: mustBig("264665000000000000000000"),
	})
	require.NoError(t, err)
	require.Zero(t, before.DeltaSpecified.Sign())
	require.Zero(t, before.DeltaUnspecified.Sign())

	after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: false},
		AmountOut:        mustBig("16019790377638234"),
	})
	require.NoError(t, err)
	// 16019790377638234 * 30000 / 1e6 = 480593711329147.02 -> floor.
	require.Equal(t, "480593711329147", after.HookFee.String())
}

// TestCalcOut_BuyQuoteSkimsInput mirrors a 0.01 WETH buy: 3% off the input
// in beforeSwap, nothing in afterSwap.
func TestCalcOut_BuyQuoteSkimsInput(t *testing.T) {
	t.Parallel()
	h := feeHook()

	before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut: true, ZeroForOne: true,
		AmountSpecified: mustBig("10000000000000000"),
	})
	require.NoError(t, err)
	require.Equal(t, "300000000000000", before.DeltaSpecified.String())
	require.Zero(t, before.DeltaUnspecified.Sign())

	after, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true},
		AmountIn:         mustBig("10000000000000000"),
		AmountOut:        mustBig("159187442666741124038656"),
	})
	require.NoError(t, err)
	require.Zero(t, after.HookFee.Sign())
}

// TestCalcIn_ExactOutputGrossUp mirrors _feeFor gross-up: naming 1 WETH out
// adds 1e18*30000/970000 on the specified side.
func TestCalcIn_ExactOutputGrossUp(t *testing.T) {
	t.Parallel()
	h := feeHook()

	before, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{
		CalcOut: false, ZeroForOne: false,
		AmountSpecified: mustBig("1000000000000000000"),
	})
	require.NoError(t, err)
	// 1e18 * 30000 / 970000 = 30927835051546391.75 -> floor.
	require.Equal(t, "30927835051546391", before.DeltaSpecified.String())
}

// TestUntrackedFeeRateFailsLoud: without a tracked rate the hook refuses to
// quote instead of silently quoting zero-fee.
func TestUntrackedFeeRateFailsLoud(t *testing.T) {
	t.Parallel()
	h := &Hook{Hook: &uniswapv4.BaseHook{Exchange: valueobject.ExchangeUniswapV4Cashcat}}

	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true,
		AmountSpecified: big.NewInt(1000)})
	require.ErrorIs(t, err, ErrNotCalibrated)

	_, err = h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountOut:        big.NewInt(1000),
	})
	require.ErrorIs(t, err, ErrNotCalibrated)
}

// TestTrack_PassthroughAndNilRPC: own-schema HookExtra returns normalized;
// missing extra without RPC errors instead of panicking.
func TestTrack_PassthroughAndNilRPC(t *testing.T) {
	t.Parallel()

	h := feeHook()
	out, err := h.Track(t.Context(), &uniswapv4.HookParam{
		HookExtra: uniswapv4.HookExtra(`{"f":30000}`),
	})
	require.NoError(t, err)
	require.JSONEq(t, `{"f":30000}`, string(out))

	_, err = (&Hook{}).Track(t.Context(), &uniswapv4.HookParam{})
	require.ErrorIs(t, err, ErrNotCalibrated)
}

// TestTrack_RefetchesOnForeignSchema: the auto-detection model persisted
// before explicit registration must NOT be adopted -- without an "f" rate
// the hook falls through to an on-chain read (nil RPC here), so the first
// track after deploy migrates the pool instead of freezing the stale fit.
func TestTrack_RefetchesOnForeignSchema(t *testing.T) {
	t.Parallel()

	autoModel := `{"f":[{},{"s":[{"x":53.07,"o":0.147}]}],"m":["688502902323479103",null],"b":64}`
	_, err := (&Hook{}).Track(t.Context(), &uniswapv4.HookParam{
		HookAddress: common.HexToAddress("0x75a54357d9c78a2db19004a5fdc76c50f9242aec"),
		HookExtra:   uniswapv4.HookExtra(autoModel),
	})
	require.ErrorIs(t, err, ErrNotCalibrated,
		"foreign schema must trigger refetch (nil RPC here), never passthrough")
}

func mustBig(s string) *big.Int {
	v, ok := new(big.Int).SetString(s, 10)
	if !ok {
		panic(s)
	}
	return v
}

// TestEndToEnd_SellQuotesThroughPoolSimulator builds the real pool simulator
// for 0x6f0e... (Robinhood snapshot, HookExtra already migrated to {"f":30000})
// and asserts the previously-banned size quotes at raw*0.97: no Min bound,
// no fitted spline, exactly the flat 3% the hook charges on-chain.
func TestEndToEnd_SellQuotesThroughPoolSimulator(t *testing.T) {
	t.Parallel()

	extra := `{"liquidity":36819258015569838458222,"sqrtPriceX96":321747665048613514693004700497438,"tickSpacing":200,"tick":166192,"ticks":[{"index":-887200,"liquidityGross":36819258015569838458222,"liquidityNet":36819258015569838458222},{"index":204200,"liquidityGross":36819258015569838458222,"liquidityNet":-36819258015569838458222}],"hX":{"f":30000}}`
	ep := entity.Pool{
		Address:  "0x6f0e1919cb58a4b617e28e9be011c2fe4cb499663e89a4bb5ef7e9e487438383",
		Exchange: string(valueobject.ExchangeUniswapV4Cashcat),
		Type:     "uniswap-v4",
		Reserves: entity.PoolReserves{"7710832767622379520", "149523981364570752796327936"},
		Tokens: []*entity.PoolToken{
			{Address: "0x0bd7d308f8e1639fab988df18a8011f41eacad73", Symbol: "WETH", Decimals: 18, Swappable: true},
			{Address: "0x65fa36fe3c0f4beb9d793cd9a79c7f53ef4b82cc", Symbol: "PURRS", Decimals: 18, Swappable: true},
		},
		Extra:       extra,
		StaticExtra: `{"0x0":[true,false],"fee":0,"tS":200,"hooks":"0x75a54357d9c78a2db19004a5fdc76c50f9242aec","uR":"0x8876789976decbfcbbbe364623c63652db8c0904","pm2":"0x000000000022d473030f116ddee9f6b43ac78ba3","mc3":"0x2cac2d899ecc914d704feaae33ac1bf36277dad1"}`,
	}
	sim, err := uniswapv4.NewPoolSimulator(ep, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	require.Equal(t, string(valueobject.ExchangeUniswapV4Cashcat), sim.GetExchange())

	out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x65fa36fe3c0f4beb9d793cd9a79c7f53ef4b82cc",
			Amount: mustBig("264665000000000000000000"),
		},
		TokenOut: "0x0bd7d308f8e1639fab988df18a8011f41eacad73",
	})
	require.NoError(t, err, "previously banned size must quote once migrated")
	// raw AMM ~16019790377638234; flat 3% -> ~15539196666309087, matching the
	// Tenderly execution intermediate. Allow 1% tolerance for state drift.
	got, _ := new(big.Float).SetInt(out.TokenAmountOut.Amount).Float64()
	require.InDelta(t, 15539196666309087.0, got, 15539196666309087.0*0.01)
}
