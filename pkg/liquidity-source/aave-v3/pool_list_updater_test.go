package aavev3

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestPoolListUpdater(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	plUpdater := NewPoolsListUpdater(&Config{
		DexID:       DexType,
		PoolAddress: "0x87870bca3f3fd6335c3f4ce8392d69350b4fa4e2",
	}, ethrpc.New("https://ethereum.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")))

	newPools, _, err := plUpdater.GetNewPools(context.Background(), nil)
	require.NoError(t, err)
	require.Greater(t, len(newPools), 0)
}
