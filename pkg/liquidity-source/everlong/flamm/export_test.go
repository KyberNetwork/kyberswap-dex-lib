package everlongflamm

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Test-only hooks for the external test package (msgpack_external_test.go), which imports
// pkg/msgpack -- the production encoder, whose generated registry imports this package.

// ExportEntity is a tracked entity over a core end-to-end scenario state.
func ExportEntity(t *testing.T, block, tag string, policy Policy) entity.Pool {
	return simEntity(t, gridReads(t, block, tag), policy)
}

// ExportSetClock pins a simulator's quote clock (unexported, never serialized).
func ExportSetClock(p *PoolSimulator, ts uint64) {
	p.nowFn = func() uint64 { return ts }
}

// ExportTimestamp is the adopted state's timestamp.
func ExportTimestamp(p *PoolSimulator) uint64 {
	return p.state.Timestamp
}

// ExportSeq exposes the transition counter and the broken flag.
func ExportSeq(p *PoolSimulator) (uint64, bool) {
	return p.seq, p.broken
}

// ExportStateJSON renders the state.
func ExportStateJSON(t *testing.T, p *PoolSimulator) string {
	raw, err := json.Marshal(p.state)
	require.NoError(t, err)
	return string(raw)
}

// ExportNextJSON renders a SwapInfo's post-state.
func ExportNextJSON(t *testing.T, r *pool.CalcAmountOutResult) string {
	raw, err := json.Marshal(r.SwapInfo.(SwapInfo).next)
	require.NoError(t, err)
	return string(raw)
}

// ExportCalcVenue quotes on one venue (-1: the simulator's choice).
func ExportCalcVenue(p *PoolSimulator, params pool.CalcAmountOutParams, venue int) (*pool.CalcAmountOutResult, error) {
	return p.calcAmountOut(params, venue)
}
