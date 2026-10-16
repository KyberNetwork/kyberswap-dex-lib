package saturnusdat

import (
	"context"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

const testRPC = "https://ethereum-rpc.publicnode.com"

func newTestTracker() *PoolTracker {
	client := ethrpc.New(testRPC).
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	return &PoolTracker{config: &Config{DexID: DexType}, ethrpcClient: client}
}

func listedPool(t *testing.T) entity.Pool {
	t.Helper()
	pools, _, err := NewPoolsListUpdater(&Config{DexID: DexType}, nil).GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	return pools[0]
}

func TestPoolTracker_Latest(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	p, err := newTestTracker().GetNewPoolState(context.Background(), listedPool(t), pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.Greater(t, p.BlockNumber, uint64(26197865))

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	t.Logf("cap=%s balance=%s reserves=%v", sim.AssetCap.Dec(), sim.AssetBalance.Dec(), p.Reserves)

	res, err := quote(sim, USDC, USDat, "1000000")
	require.NoError(t, err)
	assert.Equal(t, "1000000", res.TokenAmountOut.Amount.String())
}

func TestPoolTracker_BadStaticExtra(t *testing.T) {
	t.Parallel()
	p := listedPool(t)
	p.StaticExtra = "x"
	_, err := newTestTracker().GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
	assert.Error(t, err)
}

func TestPoolTracker_ExecutorWhitelist(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	tracker := newTestTracker()
	tracker.config.Executor = "0x000000000000000000000000000000000000dead"
	p, err := tracker.GetNewPoolState(context.Background(), listedPool(t), pool.GetNewPoolStateParams{})
	require.NoError(t, err)

	var extra Extra
	require.NoError(t, json.Unmarshal([]byte(p.Extra), &extra))
	require.True(t, extra.WhitelistEnabled)
	assert.True(t, extra.NotWhitelisted, "an arbitrary address is not whitelisted")
	assert.Equal(t, entity.PoolReserves{"0", "0"}, p.Reserves)
}
