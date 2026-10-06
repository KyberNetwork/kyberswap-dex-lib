package orderbook

import (
	"math"
	"testing"
	"time"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestMakerBookExpiry(t *testing.T) {
	ep := entityPool
	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(ep.Extra), &extra))
	extra.ValidUntil = time.Now().Unix() - 1
	raw, err := json.Marshal(extra)
	require.NoError(t, err)
	ep.Extra = string(raw)
	// Refetching a stale book cannot freshen it through the pool timestamp.
	ep.Timestamp = time.Now().Unix()
	_, err = NewPoolSimulatorWith(ep, math.MaxInt64)
	require.ErrorIs(t, err, ErrLevelsTooOld)
	extra.ValidUntil = time.Now().Unix() + 60
	raw, err = json.Marshal(extra)
	require.NoError(t, err)
	ep.Extra = string(raw)
	sim, err := NewPoolSimulatorWith(ep, MaxAge)
	require.NoError(t, err)
	cloned := sim.CloneState().(*PoolSimulator)
	require.Equal(t, sim.validUntil, cloned.validUntil)
	sim.validUntil = time.Now().Unix() - 1
	_, err = sim.CalcAmountOut(pool.CalcAmountOutParams{})
	require.ErrorIs(t, err, ErrLevelsTooOld)
	_, err = sim.CalcAmountIn(pool.CalcAmountInParams{})
	require.ErrorIs(t, err, ErrLevelsTooOld)
	extra.ValidUntil = 0
	raw, err = json.Marshal(extra)
	require.NoError(t, err)
	ep.Extra = string(raw)
	_, err = NewPoolSimulatorWith(ep, MaxAge)
	require.NoError(t, err)
}

func TestMinimumRawAmount(t *testing.T) {
	for _, tc := range []struct {
		size     float64
		decimals uint8
		want     string
	}{
		{0.5, 18, "500000000000000000"}, {0.005, 18, "5000000000000000"}, {0.0000001, 6, "1"}, {1.2345671, 6, "1234568"},
	} {
		require.Equal(t, tc.want, minimumRawAmount(tc.size, tc.decimals).String())
	}
}
