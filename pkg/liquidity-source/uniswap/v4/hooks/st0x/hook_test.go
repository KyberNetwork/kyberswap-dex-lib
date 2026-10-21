package st0x

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestHookFactoryUsesPoolReserves(t *testing.T) {
	for name, extra := range map[string]string{
		"missing": `{"p":1000000000000000000}`,
		"stale":   `{"p":1000000000000000000,"r0":"1","r1":"2"}`,
	} {
		t.Run(name, func(t *testing.T) {
			hook, ok := uniswapv4.GetHook(hookAddress, &uniswapv4.HookParam{
				Pool: &entity.Pool{Reserves: entity.PoolReserves{"1000", "2000"}}, HookExtra: uniswapv4.HookExtra(extra),
			})
			require.True(t, ok)
			h := hook.(*Hook)
			require.Equal(t, "1000", h.Reserve0.Dec())
			require.Equal(t, "2000", h.Reserve1.Dec())
			// Preserve the existing HookExtra JSON fields for older consumers.
			encoded, err := json.Marshal(h)
			require.NoError(t, err)
			var restored Hook
			require.NoError(t, json.Unmarshal(encoded, &restored))
			require.Equal(t, h.Reserve0, restored.Reserve0)
			require.Equal(t, h.Reserve1, restored.Reserve1)
			for _, direction := range []bool{true, false} {
				for _, calcOut := range []bool{true, false} {
					result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: direction, AmountSpecified: big.NewInt(10)})
					require.NoError(t, err)
					require.Equal(t, "1000", h.Reserve0.Dec())
					require.Equal(t, "2000", h.Reserve1.Dec())
					clone := h.CloneState().(*Hook)
					clone.UpdateBalance(result.SwapInfo)
					if direction {
						require.Equal(t, "1010", clone.Reserve0.Dec())
						require.Equal(t, "1990", clone.Reserve1.Dec())
					} else {
						require.Equal(t, "990", clone.Reserve0.Dec())
						require.Equal(t, "2010", clone.Reserve1.Dec())
					}
					require.Equal(t, "1000", h.Reserve0.Dec())
					require.Equal(t, "2000", h.Reserve1.Dec())
				}
			}
			_, err = h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(2001)})
			require.ErrorIs(t, err, ErrInsufficientRsv)
		})
	}
}

func TestPoolSimulatorWithMissingHookExtraReserves(t *testing.T) {
	const token0 = "0x0000000000000000000000000000000000000001"
	const token1 = "0x0000000000000000000000000000000000000002"
	p := entity.Pool{
		Address:     "0xe26507ae709887aa01dab0e959ee4b3a8a1c025e5c29e1b2644b62ea950a4253",
		Reserves:    entity.PoolReserves{"1000", "2000"},
		Tokens:      []*entity.PoolToken{{Address: token0, Decimals: 18}, {Address: token1, Decimals: 18}},
		Extra:       `{"liquidity":0,"sqrtPriceX96":79228162514264337593543950336,"tick":0,"tickSpacing":1,"ticks":[],"hX":{"p":1000000000000000000}}`,
		StaticExtra: `{"hooks":"0x9af1a97021b92c47219d2fb24deba51c248a2a88","fee":0,"tS":1}`,
	}
	sim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainID(8453))
	require.NoError(t, err)
	params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(10)}, TokenOut: token1}
	for i := 0; i < 2; i++ {
		result, err := sim.CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, big.NewInt(10), result.TokenAmountOut.Amount)
	}
}

func TestBeforeSwapMissingReserves(t *testing.T) {
	for _, reserves := range [][2]*uint256.Int{{nil, nil}, {nil, uint256.NewInt(100)}, {uint256.NewInt(100), nil}} {
		h := &Hook{Price: 1e18, Reserve0: reserves[0], Reserve1: reserves[1]}
		for _, direction := range []bool{true, false} {
			for _, calcOut := range []bool{true, false} {
				_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: calcOut, ZeroForOne: direction, AmountSpecified: big.NewInt(10)})
				require.ErrorIs(t, err, ErrMissingReserves)
			}
		}
	}
}

// The contract reverts when block.timestamp > updatedAt + maxStaleness (seconds), so the gate must
// compare quote-clock seconds, with a margin for the quote-to-execution gap. The old gate compared
// a block number with unix seconds and never refused.
func TestBeforeSwapStalenessGate(t *testing.T) {
	const updatedAt, maxStaleness = uint64(1_800_000_000), uint64(60)

	for name, tc := range map[string]struct {
		now     int64
		maxStal uint64
		stale   bool
	}{
		"fresh":                       {int64(updatedAt) + 10, maxStaleness, false},
		"just before margin":          {int64(updatedAt+maxStaleness) - stalenessMarginSec, maxStaleness, false},
		"inside margin":               {int64(updatedAt+maxStaleness) - stalenessMarginSec + 1, maxStaleness, true},
		"past deadline":               {int64(updatedAt+maxStaleness) + 1, maxStaleness, true},
		"tracker dead for a day":      {int64(updatedAt) + 86_400, maxStaleness, true},
		"no staleness limit on-chain": {int64(updatedAt) + 86_400, 0, false},
	} {
		t.Run(name, func(t *testing.T) {
			h := &Hook{
				Price: 1e18, UpdatedAt: updatedAt, MaxStaleness: tc.maxStal,
				Reserve0: uint256.NewInt(1000), Reserve1: uint256.NewInt(1000),
				quoteAt: tc.now,
			}
			_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(10)})
			if tc.stale {
				require.ErrorIs(t, err, ErrStalePrice)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// Routing builds carry no snapshot clock, so the gate keeps reading the wall clock.
func TestBeforeSwapStalenessGateUsesWallClockWhenUnset(t *testing.T) {
	h := &Hook{
		Price: 1e18, UpdatedAt: 1, MaxStaleness: 1,
		Reserve0: uint256.NewInt(1000), Reserve1: uint256.NewInt(1000),
	}
	_, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(10)})
	require.ErrorIs(t, err, ErrStalePrice)
}

// hX must stay pure chain state: no tracker clock, or the tracker's activity bump would fire on
// every refresh.
func TestHookExtraHasNoTrackerClock(t *testing.T) {
	b, err := json.Marshal(&Hook{Price: 1, UpdatedAt: 2, MaxStaleness: 3})
	require.NoError(t, err)
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	require.NotContains(t, m, "t")
}
