package upshiftusdf

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/test"
)

// UPSHIFT_USDF_RPC overrides the RPC, e.g. an anvil fork.
func TestPoolTracker_Latest(t *testing.T) {
	t.Parallel()
	test.SkipCI(t)

	rpc := os.Getenv("UPSHIFT_USDF_RPC")
	if rpc == "" {
		rpc = "https://ethereum-rpc.publicnode.com"
	}
	client := ethrpc.New(rpc).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	tracker := &PoolTracker{config: &Config{DexID: DexType}, ethrpcClient: client}
	pools, _, err := NewPoolsListUpdater(&Config{DexID: DexType}, nil).GetNewPools(context.Background(), nil)
	require.NoError(t, err)

	p, err := tracker.GetNewPoolState(context.Background(), pools[0], pool.GetNewPoolStateParams{})
	require.NoError(t, err)
	assert.Greater(t, p.BlockNumber, uint64(26204000))

	sim, err := NewPoolSimulator(p)
	require.NoError(t, err)
	t.Logf("%+v reserves=%v", sim.Extra, p.Reserves)
	assert.EqualValues(t, startOfYear, sim.StartOfYear)
	assert.Equal(t, "100000", sim.MinDeposit.Dec())
	assert.Equal(t, "10000000000000", sim.MaxSupply.Dec())

	res, err := quote(sim, "1000000000")
	require.NoError(t, err)
	t.Logf("quote %s gas %d", res.TokenAmountOut.Amount, res.Gas)
}

func TestPoolTracker_BadTokens(t *testing.T) {
	t.Parallel()
	tracker := &PoolTracker{config: &Config{DexID: DexType}, ethrpcClient: ethrpc.New("http://127.0.0.1:1")}
	pools, _, _ := NewPoolsListUpdater(&Config{DexID: DexType}, nil).GetNewPools(context.Background(), nil)
	_, err := tracker.GetNewPoolState(context.Background(), pools[0], pool.GetNewPoolStateParams{})
	assert.Error(t, err)
}
