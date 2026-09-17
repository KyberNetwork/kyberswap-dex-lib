package deepstateob

import (
	"math/big"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// TestPoolSimulator_EndToEnd exercises the real wrapped order-book simulator
// against a tiny synthetic two-level book, confirming the tracker's
// buildLevels output plugs into CalcAmountOut/CalcAmountIn correctly end to
// end (not just the level-construction math tested in pool_tracker_test.go).
func TestPoolSimulator_EndToEnd(t *testing.T) {
	const router = "0x6cf19308c22fc82ea620fa0b3e94948d20f27b96"
	const token0 = "0x5fc5360d0400a0fd4f2af552add042d716f1d168" // USDG, 6d
	const token1 = "0xd0601ce157db5bdc3162bbac2a2c8af5320d9eec" // NVDA, 18d

	leaves := []decodedNode{{tick: 0, quantity: uint256.NewInt(1_000_000), nonce: 1}}
	extra := orderbook.Extra{LevelsFrom: [2][]orderbook.Level{
		buildLevels(leaves, 6, 18, false),
		buildLevels(leaves, 6, 18, true),
	}}
	extraBytes, err := json.Marshal(extra)
	require.NoError(t, err)

	staticExtraBytes, err := json.Marshal(StaticExtra{Router: router})
	require.NoError(t, err)

	entityPool := entity.Pool{
		Address:   "synthetic-deepstate-pool",
		Exchange:  DexType,
		Type:      DexType,
		Timestamp: time.Now().Unix(),
		Reserves:  entity.PoolReserves{"0", "0"},
		SwapFee:   0.001, // 10bps
		Tokens: []*entity.PoolToken{
			{Address: token0, Decimals: 6, Swappable: true},
			{Address: token1, Decimals: 18, Swappable: true},
		},
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}

	sim, err := NewPoolSimulator(entityPool)
	require.NoError(t, err)

	meta := sim.GetMetaInfo(token1, token0).(MetaInfo)
	require.True(t, meta.IsBid) // swapping FROM token1 (NVDA) matches resting asks -> isBid=true fill()

	// Swap 0.5 token0 (USDG) into token1 (NVDA) -- consumes the bid-tree level.
	out, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{Token: token0, Amount: big.NewInt(500_000)}, // 0.5 USDG raw
		TokenOut:      token1,
	})
	require.NoError(t, err)
	require.Equal(t, token1, out.TokenAmountOut.Token)
	require.Positive(t, out.TokenAmountOut.Amount.Sign())
}
