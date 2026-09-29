package mofo

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const launchedAt = int64(1790284031) // MOFO's pools, 2026-09-24T21:07:11Z

// currentFeePips must reproduce MofoHook.currentFeePips bit for bit:
// fee = 990000 - (990000 - feePips) * elapsed / 3 (floored) for elapsed < 3, else feePips.
func TestCurrentFeePips_LaunchWindow(t *testing.T) {
	for _, c := range []struct {
		elapsed int64
		want    uint32
	}{
		{0, 990_000},     // the launch second
		{1, 666_667},     // 990000 - 970000/3 = 990000 - 323333
		{2, 343_334},     // 990000 - 1940000/3 = 990000 - 646666
		{3, 20_000},      // settled: the pool's own fee from 3 s on
		{4, 20_000},      //
		{86_400, 20_000}, //
		{-5, 990_000},    // a clock behind the launch reads as the launch second (most expensive)
	} {
		assert.Equalf(t, c.want, currentFeePips(20_000, launchedAt, launchedAt+c.elapsed), "elapsed %d", c.elapsed)
	}
	// the 20% ceiling pool: 990000 - 790000*1/3 = 726667 at 1 s
	assert.Equal(t, uint32(726_667), currentFeePips(200_000, launchedAt, launchedAt+1))
}

func TestBeforeSwap_ReturnsOverrideFeeOnly(t *testing.T) {
	orig := NowFn
	defer func() { NowFn = orig }()

	h := &Hook{Hook: &uniswapv4.BaseHook{}, Extra: Extra{FeePips: 20_000, LaunchedAt: launchedAt}}

	NowFn = func() int64 { return launchedAt + 3600 }
	res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: bignumber.TenPowInt(18)})
	require.NoError(t, err)
	require.NoError(t, uniswapv4.ValidateBeforeSwapResult(res))
	assert.Equal(t, uniswapv4.FeeAmount(20_000), res.SwapFee)
	assert.Zero(t, res.DeltaSpecified.Sign(), "no delta: the fee is an LP fee")
	assert.Zero(t, res.DeltaUnspecified.Sign())

	NowFn = func() int64 { return launchedAt + 1 }
	res, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, ZeroForOne: false, AmountSpecified: bignumber.TenPowInt(18)})
	require.NoError(t, err)
	assert.Equal(t, uniswapv4.FeeAmount(666_667), res.SwapFee)
}

// A pool Track never read must not be priced: slot0.lpFee is 0 on every mofo pool, so pricing it anyway would
// quote a fee-free swap that the hook would then charge 2%+ on.
func TestBeforeSwap_UntrackedPoolRefusesToQuote(t *testing.T) {
	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: bignumber.TenPowInt(18)})
	assert.ErrorIs(t, err, ErrPoolNotRegistered)

	h.Extra = Extra{FeePips: maxFeePips + 1, LaunchedAt: launchedAt}
	_, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: bignumber.TenPowInt(18)})
	assert.ErrorIs(t, err, ErrInvalidFee)
}

// afterSwap takes no hook fee: the base hook's zero result applies.
func TestAfterSwap_NoHookFee(t *testing.T) {
	h := &Hook{Hook: &uniswapv4.BaseHook{}, Extra: Extra{FeePips: 20_000, LaunchedAt: launchedAt}}
	res, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountIn:         bignumber.TenPowInt(18),
		AmountOut:        bignumber.TenPowInt(18),
	})
	require.NoError(t, err)
	assert.Zero(t, res.HookFee.Sign())
}

// The registered factory rebuilds the hook from the tracked extra, and the address's permission bits select
// beforeSwap and afterSwap only (0x28c0: beforeInitialize | beforeAddLiquidity | beforeSwap | afterSwap).
func TestFactory_RegisteredAndPermissions(t *testing.T) {
	addr := common.HexToAddress("0x665C52D02Ddc506dfb3158C100Edc77FE412a8c0")
	hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(`{"f":20000,"l":1790284031}`)})
	require.True(t, ok)
	h, isMofo := hook.(*Hook)
	require.True(t, isMofo)
	assert.Equal(t, Extra{FeePips: 20_000, LaunchedAt: launchedAt}, h.Extra)
	assert.Equal(t, string(valueobject.ExchangeUniswapV4Mofo), h.GetExchange())
	assert.True(t, h.CanBeforeSwap(addr))
	assert.True(t, h.CanAfterSwap(addr))
	assert.True(t, uniswapv4.HasSwapPermissions(addr))
}
