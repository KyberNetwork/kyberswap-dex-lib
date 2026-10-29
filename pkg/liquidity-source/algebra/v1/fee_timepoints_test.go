package algebrav1

import (
	"encoding/json"
	"math/big"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra"
)

// feeFixture is a file in testdata/fee: a pool at one block and the fee a swap paid in that block on chain.
type feeFixture struct {
	Block                     uint64               `json:"block"`
	BlockTimestamp            uint32               `json:"blockTimestamp"`
	Tick                      int64                `json:"tick"`
	TimepointIndex            uint16               `json:"timepointIndex"`
	Liquidity                 *big.Int             `json:"liquidity"`
	FeeConfig                 FeeConfiguration     `json:"feeConfig"`
	VolumePerLiquidityInBlock *big.Int             `json:"volumePerLiquidityInBlock"`
	Timepoints                map[uint16]Timepoint `json:"timepoints"`
	StoredFee                 uint16               `json:"storedFee"`
	SwapFee                   uint16               `json:"swapFee"`
}

func readJSON[T any](t *testing.T, file string) T {
	t.Helper()
	b, err := os.ReadFile(file)
	require.NoError(t, err)
	var v T
	require.NoError(t, json.Unmarshal(b, &v))
	return v
}

// TestFeeFromFetchedTimepoints: each fixture holds what the tracker fetches for a pool at one block (the current
// timepoint, the one before it, the oldest candidates), the pool's state, fee configuration and volume, and the fee a
// swap paid in that block on chain. The pools' last swaps are days to years old. Kept as the tracker prunes them, the
// timepoints give the fee the pool swapped with.
func TestFeeFromFetchedTimepoints(t *testing.T) {
	files, err := filepath.Glob("testdata/fee/*.json")
	require.NoError(t, err)
	require.NotEmpty(t, files)

	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			fx := readJSON[feeFixture](t, file)
			// the last timepoint is over a day old, so the fetch stops at it
			require.Less(t, fx.Timepoints[fx.TimepointIndex].BlockTimestamp, fx.BlockTimestamp-WINDOW)
			algebra.PruneTimepoints(fx.Timepoints, fx.TimepointIndex, fx.TimepointIndex)

			ts := TimepointStorage{data: fx.Timepoints, updates: map[uint16]Timepoint{}}
			tick := int24(fx.Tick)
			idx, err := ts.write(fx.TimepointIndex, fx.BlockTimestamp, tick, fx.Liquidity, fx.VolumePerLiquidityInBlock)
			require.NoError(t, err)
			fee, err := ts._getNewFee(fx.BlockTimestamp, tick, idx, fx.Liquidity, &fx.FeeConfig)
			require.NoError(t, err)
			assert.Equal(t, fx.SwapFee, fee, "the pool stores %d from its last swap", fx.StoredFee)
		})
	}
}
