package uniswapv4

import (
	"context"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
)

// extraSpyHook records the pool state that Track observes.
type extraSpyHook struct {
	*BaseHook
	trackedExtra Extra
}

func (h *extraSpyHook) Track(_ context.Context, param *HookParam) (json.RawMessage, error) {
	if err := json.Unmarshal([]byte(param.Pool.Extra), &h.trackedExtra); err != nil {
		return nil, err
	}
	return json.RawMessage(`{"new":true}`), nil
}

// Track's RPC calls observe the block just fetched, so the Pool.Extra it sees
// must describe that same block. A hook that simulates the pool inside Track
// (the auto-detect hook inverts its quoter probes against a simulator built
// from Pool.Extra) would otherwise read a price move since the previous round
// as a hook fee, and attribute it to the wrong swap side.
func TestResolveHookState_TrackSeesFreshPoolState(t *testing.T) {
	sqrtPriceOne, _ := new(big.Int).SetString("79228162514264337593543950336", 10)
	ticks := func(l int64) []Tick {
		return []Tick{
			{Index: -600, LiquidityGross: big.NewInt(l), LiquidityNet: big.NewInt(l)},
			{Index: 600, LiquidityGross: big.NewInt(l), LiquidityNet: big.NewInt(-l)},
		}
	}

	staleExtra, err := json.Marshal(Extra{
		Extra: &uniswapv3.Extra{
			Liquidity:    big.NewInt(1e18),
			SqrtPriceX96: sqrtPriceOne,
			TickSpacing:  60,
			Tick:         big.NewInt(0),
			Ticks:        ticks(1e18),
		},
	})
	require.NoError(t, err)
	p := &entity.Pool{Address: "0xpool", Extra: string(staleExtra)}

	freshSqrtPrice, _ := new(big.Int).SetString("79267784519130042428790663799", 10) // price +0.1%
	freshTicks := ticks(2e18)
	result := &FetchRPCResult{
		Liquidity:   big.NewInt(2e18),
		TickSpacing: 60,
		Slot0:       Slot0Data{SqrtPriceX96: freshSqrtPrice, Tick: big.NewInt(9)},
	}

	hook := &extraSpyHook{BaseHook: &BaseHook{}}
	tracker := &PoolTracker{}
	require.NoError(t, tracker.resolveHookState(context.Background(), hook, &HookParam{Pool: p}, result, freshTicks))

	got := hook.trackedExtra
	require.NotNil(t, got.Extra)
	assert.Equal(t, freshSqrtPrice.String(), got.SqrtPriceX96.String(), "Track saw the previous round's sqrtPriceX96")
	assert.Equal(t, "9", got.Tick.String(), "Track saw the previous round's tick")
	assert.Equal(t, big.NewInt(2e18).String(), got.Liquidity.String(), "Track saw the previous round's liquidity")
	require.Len(t, got.Ticks, len(freshTicks))
	for i := range freshTicks {
		assert.Equal(t, freshTicks[i].LiquidityNet.String(), got.Ticks[i].LiquidityNet.String(), "Track saw the previous round's ticks")
	}
	assert.JSONEq(t, `{"new":true}`, string(result.HookExtra))
}

// The tracker persists Track's output through ToExtra. If it dropped hX,
// every hook's carried state would silently reset on each refresh.
func TestFetchRPCResult_ToExtra_CarriesHookExtra(t *testing.T) {
	result := &FetchRPCResult{
		Liquidity:   big.NewInt(7),
		TickSpacing: 60,
		Slot0:       Slot0Data{SqrtPriceX96: big.NewInt(11), Tick: big.NewInt(-3)},
		HookExtra:   json.RawMessage(`{"model":1}`),
	}
	ticks := []Tick{{Index: -60, LiquidityGross: big.NewInt(7), LiquidityNet: big.NewInt(7)}}

	buf, err := json.Marshal(result.ToExtra(ticks))
	require.NoError(t, err)
	var got Extra
	require.NoError(t, json.Unmarshal(buf, &got))

	require.NotNil(t, got.Extra)
	assert.Equal(t, "7", got.Liquidity.String())
	assert.Equal(t, uint64(60), got.TickSpacing)
	assert.Equal(t, "11", got.SqrtPriceX96.String())
	assert.Equal(t, "-3", got.Tick.String())
	require.Len(t, got.Ticks, 1)
	assert.Equal(t, -60, got.Ticks[0].Index)
	assert.JSONEq(t, `{"model":1}`, string(got.HookExtra), "Track's output must be persisted")
}
