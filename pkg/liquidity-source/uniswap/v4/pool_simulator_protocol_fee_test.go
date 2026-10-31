package uniswapv4

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// The expected amounts below are what v4-core's PoolManager.swap returns for the same pool:
// full range (tickSpacing 60), sqrtPriceX96 = 2^96, liquidity 1e24, lpFee 3000, exact-in 1e18.

const (
	pfToken0 = "0x0000000000000000000000000000000000000001"
	pfToken1 = "0x0000000000000000000000000000000000000002"
)

// overrideHookAddr has only the BeforeSwap permission (bit 7); the hook returns the LP fee
// override stored in its hook extra.
var overrideHookAddr = common.HexToAddress("0x00000000000000000000000000000000000d0080")

type lpFeeOverrideHook struct {
	*BaseHook
	Fee FeeAmount `json:"fee"`
}

func (h *lpFeeOverrideHook) BeforeSwap(*BeforeSwapParams) (*BeforeSwapResult, error) {
	return &BeforeSwapResult{DeltaSpecified: new(big.Int), DeltaUnspecified: new(big.Int), SwapFee: h.Fee}, nil
}

func (h *lpFeeOverrideHook) CloneState() Hook { return h }

var _ = RegisterHooksFactory(func(param *HookParam) Hook {
	h := &lpFeeOverrideHook{BaseHook: &BaseHook{}}
	_ = param.HookExtra.Unmarshal(h)
	return h
}, overrideHookAddr)

func packProtocolFee(zeroForOne, oneForZero uint32) uint32 { return oneForZero<<12 | zeroForOne }

// newProtocolFeeTestPool builds the pool the way the tracker stores it for the given slot0.
func newProtocolFeeTestPool(t *testing.T, slot0 Slot0Data, staticFee uint32, hook common.Address,
	hookExtra json.RawMessage) *PoolSimulator {
	t.Helper()
	L := bignumber.NewBig10("1000000000000000000000000")
	ticks := []Tick{
		{Index: -887220, LiquidityGross: new(big.Int).Set(L), LiquidityNet: new(big.Int).Set(L)},
		{Index: 887220, LiquidityGross: new(big.Int).Set(L), LiquidityNet: new(big.Int).Neg(L)},
	}
	slot0.SqrtPriceX96, slot0.Tick = new(big.Int).Lsh(big.NewInt(1), 96), big.NewInt(0)
	res := FetchRPCResult{Liquidity: L, Slot0: slot0, TickSpacing: 60, HookExtra: hookExtra}
	// same as fetchOnchainState
	res.SwapFee = uint32(CalculateSwapFee(directionalProtocolFee(slot0.ProtocolFee, true), FeeAmount(slot0.LpFee)))

	extra, err := json.Marshal(res.ToExtra(ticks))
	require.NoError(t, err)
	staticExtra, err := json.Marshal(StaticExtra{Fee: staticFee, TickSpacing: 60, HooksAddress: hook})
	require.NoError(t, err)
	reserves := bignumber.NewBig10("1000000000000000000000000000").String()

	sim, err := NewPoolSimulator(entity.Pool{
		Address:     "0x0000000000000000000000000000000000000000000000000000000000000abc",
		Exchange:    "uniswap-v4",
		Type:        DexType,
		SwapFee:     float64(res.SwapFee),
		Reserves:    entity.PoolReserves{reserves, reserves},
		Tokens:      []*entity.PoolToken{{Address: pfToken0, Decimals: 18, Swappable: true}, {Address: pfToken1, Decimals: 18, Swappable: true}},
		Extra:       string(extra),
		StaticExtra: string(staticExtra),
	}, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	return sim
}

func quoteExactIn(t *testing.T, sim *PoolSimulator, tokenIn, tokenOut string) string {
	t.Helper()
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: tokenIn, Amount: bignumber.TenPowInt(18)},
		TokenOut:      tokenOut,
	})
	require.NoError(t, err)
	return res.TokenAmountOut.Amount.String()
}

func TestPoolSimulator_ProtocolFeePerDirection(t *testing.T) {
	t.Parallel()
	const (
		outFee3000 = "996999005991991025" // swapFee 3000: no protocol fee in this direction
		outFee3997 = "996002007979012046" // swapFee 3997 = calculateSwapFee(1000, 3000)
	)

	t.Run("oneForZero fee only", func(t *testing.T) {
		sim := newProtocolFeeTestPool(t, Slot0Data{ProtocolFee: packProtocolFee(0, 1000), LpFee: 3000},
			3000, common.Address{}, nil)
		assert.Equal(t, outFee3997, quoteExactIn(t, sim, pfToken1, pfToken0))

		plain := newProtocolFeeTestPool(t, Slot0Data{LpFee: 3000}, 3000, common.Address{}, nil)
		assert.Equal(t, quoteExactIn(t, plain, pfToken0, pfToken1), quoteExactIn(t, sim, pfToken0, pfToken1))
	})

	t.Run("zeroForOne fee only", func(t *testing.T) {
		sim := newProtocolFeeTestPool(t, Slot0Data{ProtocolFee: packProtocolFee(1000, 0), LpFee: 3000},
			3000, common.Address{}, nil)
		assert.Equal(t, outFee3000, quoteExactIn(t, sim, pfToken1, pfToken0))
	})

	t.Run("exact out uses the swap's direction", func(t *testing.T) {
		sim := newProtocolFeeTestPool(t, Slot0Data{ProtocolFee: packProtocolFee(0, 1000), LpFee: 3000},
			3000, common.Address{}, nil)
		res, err := sim.CalcAmountIn(pool.CalcAmountInParams{
			TokenAmountOut: pool.TokenAmount{Token: pfToken0, Amount: bignumber.NewBig10(outFee3997)},
			TokenIn:        pfToken1,
		})
		require.NoError(t, err)
		// v4 takes exactly 1e18 token1 for this output; exact-out rounding may ask a few wei more
		diff := new(big.Int).Sub(res.TokenAmountIn.Amount, bignumber.TenPowInt(18))
		assert.True(t, diff.Sign() >= 0 && diff.Cmp(big.NewInt(10)) <= 0, "amountIn %s", res.TokenAmountIn.Amount)
	})
}

func TestPoolSimulator_HookLpFeeOverrideKeepsProtocolFee(t *testing.T) {
	t.Parallel()
	hookExtra := func(fee FeeAmount) json.RawMessage {
		b, _ := json.Marshal(map[string]FeeAmount{"fee": fee})
		return b
	}
	slot0 := Slot0Data{ProtocolFee: packProtocolFee(500, 500), LpFee: 3000}

	// override 3000: swapFee = calculateSwapFee(500, 3000) = 3499
	sim := newProtocolFeeTestPool(t, slot0, shared.DYNAMIC_FEE_FLAG, overrideHookAddr, hookExtra(3000))
	assert.Equal(t, "996500006986746537", quoteExactIn(t, sim, pfToken1, pfToken0))

	// override to 0 with OVERRIDE_FEE_FLAG: lpFee 0, swapFee = protocol fee 500
	sim = newProtocolFeeTestPool(t, slot0, shared.DYNAMIC_FEE_FLAG, overrideHookAddr, hookExtra(OverrideFeeFlag))
	assert.Equal(t, "999499001000748499", quoteExactIn(t, sim, pfToken1, pfToken0))

	// a bare 0 is no override: slot0.lpFee 3000 applies
	sim = newProtocolFeeTestPool(t, slot0, shared.DYNAMIC_FEE_FLAG, overrideHookAddr, hookExtra(0))
	assert.Equal(t, "996500006986746537", quoteExactIn(t, sim, pfToken1, pfToken0))
}

func TestFetchRPCResult_ToExtra_ProtocolFee(t *testing.T) {
	t.Parallel()
	res := FetchRPCResult{Liquidity: big.NewInt(1), Slot0: Slot0Data{SqrtPriceX96: big.NewInt(1), Tick: big.NewInt(0),
		LpFee: 3000}}

	// protocol fee off: Extra is unchanged from before pF/lpF existed
	b, err := json.Marshal(res.ToExtra(nil))
	require.NoError(t, err)
	assert.NotContains(t, string(b), `"pF"`)
	assert.NotContains(t, string(b), `"lpF"`)

	res.Slot0.ProtocolFee = packProtocolFee(0, 1000)
	b, err = json.Marshal(res.ToExtra(nil))
	require.NoError(t, err)
	var extra ExtraU256
	require.NoError(t, json.Unmarshal(b, &extra))
	assert.Equal(t, packProtocolFee(0, 1000), extra.ProtocolFee)
	assert.Equal(t, uint32(3000), extra.LpFee)
}

func TestPoolSimulator_ExtraWithoutProtocolFeeKeepsSwapFee(t *testing.T) {
	t.Parallel()
	// An Extra written before pF existed: SwapFee is used for both directions, as before.
	extra, err := json.Marshal(Extra{Extra: &uniswapv3.Extra{
		Liquidity:    bignumber.NewBig10("1000000000000000000000000"),
		SqrtPriceX96: new(big.Int).Lsh(big.NewInt(1), 96),
		TickSpacing:  60,
		Tick:         big.NewInt(0),
		Ticks: []Tick{
			{Index: -887220, LiquidityGross: bignumber.NewBig10("1000000000000000000000000"), LiquidityNet: bignumber.NewBig10("1000000000000000000000000")},
			{Index: 887220, LiquidityGross: bignumber.NewBig10("1000000000000000000000000"), LiquidityNet: bignumber.NewBig10("-1000000000000000000000000")},
		},
	}})
	require.NoError(t, err)
	staticExtra, err := json.Marshal(StaticExtra{Fee: 3000, TickSpacing: 60})
	require.NoError(t, err)
	reserves := bignumber.NewBig10("1000000000000000000000000000").String()
	sim, err := NewPoolSimulator(entity.Pool{
		Address: "0x0000000000000000000000000000000000000000000000000000000000000abc", Exchange: "uniswap-v4", Type: DexType,
		SwapFee:  3997,
		Reserves: entity.PoolReserves{reserves, reserves},
		Tokens:   []*entity.PoolToken{{Address: pfToken0, Decimals: 18, Swappable: true}, {Address: pfToken1, Decimals: 18, Swappable: true}},
		Extra:    string(extra), StaticExtra: string(staticExtra),
	}, valueobject.ChainIDEthereum)
	require.NoError(t, err)
	assert.Equal(t, "996002007979012046", quoteExactIn(t, sim, pfToken1, pfToken0))
}
