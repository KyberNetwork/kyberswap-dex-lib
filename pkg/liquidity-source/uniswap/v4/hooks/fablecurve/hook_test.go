package fablecurve

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const launchedAt = int64(1791309607) // FABLE's pools, 2026-10-06T18:00:07Z

// Every fee a Fable pool can carry: FableFactory sets 0.3% platform + 0.7% base + the creator's pick of 0% to 5% in
// whole percents, so 1% to 6% (FABLE's own pools are 1%). maxFeePips is the hook's own ceiling.
var poolFees = []uint32{10_000, 20_000, 30_000, 40_000, 50_000, 60_000}

// currentFeePips must reproduce FableHook.currentFeePips bit for bit:
// fee = 990000 - (990000 - feePips) * elapsed / 3 (floored) for elapsed < 3, else feePips.
func TestCurrentFeePips_LaunchWindow(t *testing.T) {
	for _, c := range []struct {
		feePips uint32
		want    [3]uint32 // elapsed 0, 1, 2
	}{
		{10_000, [3]uint32{990_000, 663_334, 336_667}},  // 990000 - 980000/3 = 990000 - 326666; - 1960000/3 = - 653333
		{20_000, [3]uint32{990_000, 666_667, 343_334}},  // - 970000/3 = - 323333; - 1940000/3 = - 646666
		{40_000, [3]uint32{990_000, 673_334, 356_667}},  // - 950000/3 = - 316666; - 1900000/3 = - 633333
		{60_000, [3]uint32{990_000, 680_000, 370_000}},  // - 930000/3 = - 310000; - 1860000/3 = - 620000
		{200_000, [3]uint32{990_000, 726_667, 463_334}}, // - 790000/3 = - 263333; - 1580000/3 = - 526666
	} {
		for elapsed, want := range c.want {
			assert.Equalf(t, want, currentFeePips(c.feePips, launchedAt, launchedAt+int64(elapsed)),
				"fee %d elapsed %d", c.feePips, elapsed)
		}
		for _, elapsed := range []int64{3, 4, 60, 86_400, 10 * 365 * 86_400} {
			assert.Equalf(t, c.feePips, currentFeePips(c.feePips, launchedAt, launchedAt+elapsed), "settled at %d s", elapsed)
		}
		// a clock behind the pool's creation reads as the creation second (most expensive)
		assert.Equal(t, uint32(sniperStartPips), currentFeePips(c.feePips, launchedAt, launchedAt-5))
	}
}

// A literal port of the Solidity expression, over every fee the hook accepts and the whole window.
func TestCurrentFeePips_MatchesSolidityExpression(t *testing.T) {
	for fee := uint32(1); fee <= maxFeePips; fee++ {
		for elapsed := int64(0); elapsed < 5; elapsed++ {
			want := fee
			if elapsed < sniperSeconds {
				// uint24(SNIPER_START_PIPS - (uint256(SNIPER_START_PIPS - fee) * elapsed) / SNIPER_SECONDS)
				want = uint32(new(big.Int).Sub(big.NewInt(sniperStartPips), new(big.Int).Quo(
					new(big.Int).Mul(big.NewInt(int64(sniperStartPips-fee)), big.NewInt(elapsed)),
					big.NewInt(sniperSeconds))).Uint64())
			}
			if got := currentFeePips(fee, launchedAt, launchedAt+elapsed); got != want {
				t.Fatalf("fee %d elapsed %d: got %d want %d", fee, elapsed, got, want)
			}
		}
	}
}

func TestBeforeSwap_ReturnsOverrideFeeAndCheckpointGas(t *testing.T) {
	orig := NowFn
	defer func() { NowFn = orig }()

	for _, fee := range poolFees {
		h := &Hook{Hook: &uniswapv4.BaseHook{}, Extra: Extra{FeePips: fee, LaunchedAt: launchedAt}}

		NowFn = func() int64 { return launchedAt + 3600 }
		res, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: bignumber.TenPowInt(18)})
		require.NoError(t, err)
		require.NoError(t, uniswapv4.ValidateBeforeSwapResult(res))
		assert.Equal(t, uniswapv4.FeeAmount(fee), res.SwapFee)
		assert.Zero(t, res.DeltaSpecified.Sign(), "no delta: the fee is an LP fee")
		assert.Zero(t, res.DeltaUnspecified.Sign())
		assert.Equal(t, int64(gasBeforeSwap), res.Gas, "the locker checkpoint runs on every swap")

		NowFn = func() int64 { return launchedAt }
		res, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: false, ZeroForOne: false, AmountSpecified: bignumber.TenPowInt(18)})
		require.NoError(t, err)
		assert.Equal(t, uniswapv4.FeeAmount(sniperStartPips), res.SwapFee, "the creation second: 99%%, no exemption for a router")
	}
}

// A pool Track never read must not be priced. slot0.lpFee carries the pool fee, but pricing from it alone would
// undercharge a swap in the pool's first 3 seconds (99% there), so the plugin refuses instead.
func TestBeforeSwap_UntrackedPoolRefusesToQuote(t *testing.T) {
	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: bignumber.TenPowInt(18)})
	assert.ErrorIs(t, err, ErrPoolNotRegistered)

	h.Extra = Extra{FeePips: maxFeePips + 1, LaunchedAt: launchedAt}
	_, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, AmountSpecified: bignumber.TenPowInt(18)})
	assert.ErrorIs(t, err, ErrInvalidFee)
}

// afterSwap takes no hook fee: the base hook's zero result applies (launch and mesh pools alike).
func TestAfterSwap_NoHookFee(t *testing.T) {
	h := &Hook{Hook: &uniswapv4.BaseHook{}, Extra: Extra{FeePips: 10_000, LaunchedAt: launchedAt}}
	res, err := h.AfterSwap(&uniswapv4.AfterSwapParams{
		BeforeSwapParams: &uniswapv4.BeforeSwapParams{CalcOut: true},
		AmountIn:         bignumber.TenPowInt(18),
		AmountOut:        bignumber.TenPowInt(18),
	})
	require.NoError(t, err)
	assert.Zero(t, res.HookFee.Sign())
}

// The registered factory rebuilds the hook from the tracked extra, and the address's permission bits select
// beforeSwap and afterSwap (0x38e0: beforeInitialize | afterInitialize | beforeAddLiquidity | beforeSwap | afterSwap
// | beforeDonate).
func TestFactory_RegisteredAndPermissions(t *testing.T) {
	addr := common.HexToAddress("0x376aA4C2e48AC2498CaaAc06bd7A6b6ef373f8E0")
	hook, ok := uniswapv4.GetHook(addr, &uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(`{"f":10000,"l":1791309607}`)})
	require.True(t, ok)
	h, isFable := hook.(*Hook)
	require.True(t, isFable)
	assert.Equal(t, Extra{FeePips: 10_000, LaunchedAt: launchedAt}, h.Extra)
	assert.Equal(t, string(valueobject.ExchangeUniswapV4FableCurve), h.GetExchange())
	assert.True(t, h.CanBeforeSwap(addr))
	assert.True(t, h.CanAfterSwap(addr))
	assert.True(t, uniswapv4.HasSwapPermissions(addr))

	// the flags, bit by bit (v4-core Hooks.sol: bit 13 beforeInitialize ... bit 0 afterRemoveLiquidityReturnDelta);
	// no return-delta bit (3 to 0) is set
	const (
		beforeInitialize   = 1 << 13
		afterInitialize    = 1 << 12
		beforeAddLiquidity = 1 << 11
		beforeSwap         = 1 << 7
		afterSwap          = 1 << 6
		beforeDonate       = 1 << 5
	)
	flags := (uint16(addr[18])<<8 | uint16(addr[19])) & 0x3fff
	assert.Equal(t, uint16(beforeInitialize|afterInitialize|beforeAddLiquidity|beforeSwap|afterSwap|beforeDonate), flags)

	// the zero address (the hook field of every hookless pool) is never claimed by this package
	zero, _ := uniswapv4.GetHook(common.Address{}, nil)
	_, isFable = zero.(*Hook)
	assert.False(t, isFable)
}

func TestExtraFromPoolInfo(t *testing.T) {
	birth, _ := new(big.Int).SetString("1927678248329847372080333878109930", 10) // FABLE/ETH: getSqrtPriceAtTick(202000)
	assert.Equal(t, Extra{FeePips: 10_000, LaunchedAt: launchedAt}, extraFromPoolInfo(&poolInfoRaw{
		FeePips: big.NewInt(10_000), LaunchedAt: big.NewInt(launchedAt), CoinIsCurrency1: true, BirthSqrtPriceX96: birth,
	}))
	// a mesh pool keeps the same two values: its kind changes nothing in pricing
	assert.Equal(t, Extra{FeePips: 40_000, LaunchedAt: launchedAt}, extraFromPoolInfo(&poolInfoRaw{
		FeePips: big.NewInt(40_000), LaunchedAt: big.NewInt(launchedAt), BirthSqrtPriceX96: big.NewInt(4295128740), Mesh: true,
	}))
	// never initialised by the hook: all zero, kept unpriceable
	assert.Equal(t, Extra{}, extraFromPoolInfo(&poolInfoRaw{FeePips: new(big.Int), LaunchedAt: new(big.Int), BirthSqrtPriceX96: new(big.Int)}))
	assert.Equal(t, Extra{}, extraFromPoolInfo(&poolInfoRaw{}))
}
