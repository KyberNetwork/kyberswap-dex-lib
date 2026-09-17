package curve

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestGetNewPoolState_Live decodes the hook getters on Arc mainnet. No PUMP launch is
// on its curve there yet, so it reads a CLANKER launch (graduated from birth) and
// checks it is tracked as not tradeable on the curve.
func TestGetNewPoolState_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}
	client := ethrpc.New("https://rpc.arc-scan.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tracker, err := NewPoolTracker(&Config{DexID: DexType, ChainID: valueobject.ChainIDArc, Hook: testHook}, client)
	require.NoError(t, err)

	p, err := tracker.GetNewPoolState(context.Background(), entity.Pool{
		Address:     PoolAddress("0xe9ab261f1c77caeefd37aec54859484c93c8856b95739e779e4827616cab54fe"),
		Tokens:      []*entity.PoolToken{{Address: testUsdc}, {Address: "0x43caace3d7bc72b25e32d2b81b1ee28f447e4ffd"}},
		StaticExtra: `{"hook":"` + testHook + `"}`,
	}, pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	assert.True(t, extra.Tracked)
	assert.Equal(t, uint8(1), extra.Mode)
	assert.Equal(t, uint8(statusGraduated), extra.Status)
	assert.False(t, extra.Paused)
	assert.Equal(t, int64(1), p.Timestamp)
}
