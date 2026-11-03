package machima

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3/ticks"
)

func newLensTracker(t *testing.T, useLens bool) *PoolTracker {
	t.Helper()
	tracker, err := NewPoolTracker(&Config{DexID: DexType, ClankNow: clankNowAddr,
		UseLens: useLens},
		ethrpc.New("http://127.0.0.1:0"), nil)
	require.NoError(t, err)
	return tracker
}

func trackedPool(t *testing.T) entity.Pool {
	t.Helper()
	extra := Extra{
		Liquidity:    big.NewInt(1000),
		SqrtPriceX96: big.NewInt(1 << 62),
		TickSpacing:  200,
		Tick:         big.NewInt(-200),
		Ticks: []uniswapv3.Tick{
			{Index: -400, LiquidityGross: big.NewInt(1000), LiquidityNet: big.NewInt(1000)},
			{Index: 400, LiquidityGross: big.NewInt(1000), LiquidityNet: big.NewInt(-1000)},
		},
		LensBlock: 41,
		BuyTaxBps: 100, SellTaxBps: 100, HasTax: true, PoolDeploymentTime: 7,
	}
	raw, err := json.Marshal(extra)
	require.NoError(t, err)
	return entity.Pool{Address: "0x531aae7d71343c663821604c57520b1602567006", Exchange: DexType, Type: DexType,
		Extra: string(raw), BlockNumber: 42}
}

// TestSetExtraKeepsKeysItDoesNotModel: lensDue lives in Extra outside both Extra types; rewriting
// the Machima half must not drop it, or the lens read of a pool without state could land below logs
// the pool has seen.
func TestSetExtraKeepsKeysItDoesNotModel(t *testing.T) {
	p := ticks.WithLensDue(entity.Pool{Extra: `{"liquidity":null,"tickSpacing":200}`}, nil)
	p = ticks.WithoutState(p, 1234)

	extra, err := unmarshalExtra(p.Extra)
	require.NoError(t, err)
	extra.ProtocolState = ProtocolState{HasTax: true, BuyTaxBps: 50}
	require.NoError(t, setExtra(&p, extra))

	assert.Equal(t, uint64(1234), ticks.LensDue(p))
	assert.False(t, ticks.HasState(p), "a pool without state keeps reading as one")
	got, err := unmarshalExtra(p.Extra)
	require.NoError(t, err)
	assert.True(t, got.HasTax)
	assert.Equal(t, uint64(200), got.TickSpacing)
}

// TestExtraRoundTripKeepsLensBlock: the UniV3 layer reads the logs at or below the lens block in
// Extra as late rather than adding them; losing it on the Machima rewrite would apply replayed logs
// twice.
func TestExtraRoundTripKeepsLensBlock(t *testing.T) {
	p := trackedPool(t)
	extra, err := unmarshalExtra(p.Extra)
	require.NoError(t, err)
	assert.Equal(t, uint64(41), extra.LensBlock)

	require.NoError(t, setExtra(&p, extra))
	pool, err := ticks.NewTicksBasedPool(p)
	require.NoError(t, err)
	assert.Equal(t, uint64(41), pool.LensBlock)
	assert.Len(t, pool.Ticks, 2)
}

// TestResetPoolStateKeepsTheMachimaHalf: the caller resets the outer tracker, so it must reach
// the UniV3 layer, which drops liquidity and ticks but must leave the tax config in place.
func TestResetPoolStateKeepsTheMachimaHalf(t *testing.T) {
	p, ok := newLensTracker(t, true).ResetPoolState(trackedPool(t), 50)
	require.True(t, ok)
	assert.False(t, ticks.HasState(p))
	assert.Equal(t, uint64(50), ticks.LensDue(p))

	extra, err := unmarshalExtra(p.Extra)
	require.NoError(t, err)
	assert.Empty(t, extra.Ticks)
	assert.Nil(t, extra.Liquidity)
	assert.True(t, extra.HasTax)
	assert.Equal(t, uint16(100), extra.SellTaxBps)
	assert.Equal(t, uint64(200), extra.TickSpacing)

	_, ok = newLensTracker(t, false).ResetPoolState(trackedPool(t), 50)
	assert.False(t, ok, "without the lens nothing reads the pool whole, so it is kept")
}

// TestNextUpdateAtAsksOnlyForPoolsWithoutState: the schedule only brings back a pool still without
// state, and only with the lens on.
func TestNextUpdateAtAsksOnlyForPoolsWithoutState(t *testing.T) {
	stateless := entity.Pool{Address: "0xd4829d181e93059ae602ce5a5b59ff4d6736a4a8",
		Extra: `{"liquidity":null,"tickSpacing":200}`}
	assert.Positive(t, newLensTracker(t, true).NextUpdateAt(stateless))
	assert.Zero(t, newLensTracker(t, true).NextUpdateAt(trackedPool(t)))
	assert.Zero(t, newLensTracker(t, false).NextUpdateAt(stateless))
}

// TestSetExtraClearsWhatExtraLeavesOut: a price floor gone back to nil, which Extra omits, is
// cleared from the stored Extra, while a key Extra does not model, such as the UniV3 layer's
// lensDue, is kept.
func TestSetExtraClearsWhatExtraLeavesOut(t *testing.T) {
	p := entity.Pool{Extra: `{"xmaSellSqrtPriceLimit":12345,"lensDue":77,"hasTax":true}`}
	require.NoError(t, setExtra(&p, Extra{}))

	var got map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &got))
	assert.NotContains(t, got, "xmaSellSqrtPriceLimit", "the floor is gone")
	assert.Equal(t, "77", string(got["lensDue"]), "a key Extra does not model is kept")
	assert.Equal(t, "false", string(got["hasTax"]), "a modeled key takes the new value")
}

// TestTaxBlock: the tax of a pool the lens just gave its state is read at the snapshot's block,
// which dates the pool, not at its older logs'; any other at the logs' block, or the latest.
func TestTaxBlock(t *testing.T) {
	logs := []ethtypes.Log{{BlockNumber: 90}}
	read := entity.Pool{BlockNumber: 120, Extra: `{"liquidity":1,"sqrtPriceX96":1,"ticks":[]}`}
	assert.EqualValues(t, 120, taxBlock(false, read, logs), "read whole through the lens")
	assert.EqualValues(t, 90, taxBlock(true, read, logs), "updated from its logs")
	assert.EqualValues(t, 90, taxBlock(false, entity.Pool{BlockNumber: 120, Extra: `{}`}, logs),
		"left without state")
	assert.Zero(t, taxBlock(true, read, nil), "an interval refresh, at the latest")
}
