package nadswap

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolListUpdater_Mainnet(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	cfg := &Config{
		DexID:               DexType,
		ChainID:             valueobject.ChainIDMonad,
		FactoryAddress:      "0xA25b13127e63ddae6d0b35570FF3D39dBD621001",
		FeeCollectorAddress: "0xE1C8b73343f5A83EBe165BE90470d84B00e33022",
		NewPoolLimit:        50,
	}
	client := ethrpc.New("https://rpc-mainnet.monadinfra.com/rpc/ICLJSp4IKDWLSpZ4laJATUQfL0ucwxiK").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	u := NewPoolsListUpdater(cfg, client)
	pools, _, err := u.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.NotEmpty(t, pools)

	// Classify: at least one pair should be present; ideally both meme and general.
	memeCount, generalCount := 0, 0
	for _, p := range pools {
		if p.StaticExtra == "" {
			continue
		}
		if isMeme := contains(p.StaticExtra, `"meme":true`); isMeme {
			memeCount++
		} else {
			generalCount++
		}
	}
	t.Logf("discovered %d pools (meme=%d general=%d)", len(pools), memeCount, generalCount)

	tracker, err := NewPoolTracker(cfg, client)
	require.NoError(t, err)
	for _, p := range pools[:min(3, len(pools))] {
		p2, err := tracker.GetNewPoolState(context.Background(), p, pool.GetNewPoolStateParams{})
		require.NoError(t, err)
		require.NotNil(t, p2)
		t.Logf("pool=%s block=%d", p2.Address, p2.BlockNumber)
		require.Greater(t, p2.BlockNumber, uint64(0), "tracker must populate BlockNumber from RPC fallback")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
