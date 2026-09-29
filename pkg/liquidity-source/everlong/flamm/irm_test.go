package everlongflamm

import (
	"encoding/json"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
)

// mmReadFixture decodes one testdata JSON fixture, gzipped or not (readFixture).
func mmReadFixture(t *testing.T, name string, v any) {
	t.Helper()
	require.NoError(t, json.Unmarshal(readFixture(t, "testdata/"+name), v))
}

type mmIrmRow struct {
	RateAtTarget    uint256.Int `json:"rateAtTarget"`
	Tsa             uint256.Int `json:"tsa"`
	Tba             uint256.Int `json:"tba"`
	LastUpdate      uint256.Int `json:"lastUpdate"`
	Rate            uint256.Int `json:"rate"`
	EndRateAtTarget uint256.Int `json:"endRateAtTarget"`
	Err             string      `json:"err"`
}

// TestMMIrmGrid replays AdaptiveCurveIrm.borrowRateView / borrowRate on the deployed Base IRM over
// rateAtTarget x supply x utilisation x elapsed, including the clipped wExp and the timestamp underflow.
func TestMMIrmGrid(t *testing.T) {
	t.Parallel()
	var fx struct {
		Timestamp uint64     `json:"timestamp"`
		Rows      []mmIrmRow `json:"rows"`
	}
	mmReadFixture(t, "mm_irm_grid.json.gz", &fx)
	require.Greater(t, len(fx.Rows), 1000)
	for i, r := range fx.Rows {
		m := mmMarket{TotalSupplyAssets: r.Tsa, TotalBorrowAssets: r.Tba, LastUpdate: r.LastUpdate}
		m.TotalSupplyShares.Mul(&r.Tsa, mmVirtualShares)
		m.TotalBorrowShares.Mul(&r.Tba, mmVirtualShares)
		rate, end, err := mmIrmBorrowRate(&m, &r.RateAtTarget, fx.Timestamp)
		if r.Err != "" {
			require.ErrorIs(t, err, errPanicArithmetic, "row %d", i)
			continue
		}
		require.NoError(t, err, "row %d", i)
		require.Equal(t, r.Rate.Dec(), rate.Dec(), "rate row %d %+v", i, r)
		require.Equal(t, r.EndRateAtTarget.Dec(), end.Dec(), "end row %d", i)
	}
}
