package tickerdualpool

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// Aave reserve changes emit no event, so a pool with an Aave leg must ask pool-service to
// track it again in minutes. A pool with no Aave leg needs no schedule.
func TestNextTrackAt(t *testing.T) {
	var _ uniswapv4.HookWithSchedule = &Hook{}
	require.Zero(t, (&Hook{}).NextTrackAt(nil))

	for side := range 2 {
		h := &Hook{}
		h.Reserves[side] = &AaveReserve{}
		now := time.Now().Unix()
		at := h.NextTrackAt(nil)
		require.GreaterOrEqual(t, at, now+aaveRefreshSecs)
		require.LessOrEqual(t, at, now+aaveRefreshSecs+5)
	}
}
