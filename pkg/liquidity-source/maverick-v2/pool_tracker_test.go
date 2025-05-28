package maverickv2

import (
	"context"
	"fmt"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestGetFullPoolState(t *testing.T) {
	// Create ethrpc client
	ethrpcClient := ethrpc.New("https://ethereum.kyberengineering.io").SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	// Create pool tracker
	config := &Config{
		PoolLensAddress: "0x6A9EB38DE5D349Fe751E0aDb4c0D9D391f94cc8D",
	}
	tracker, err := NewPoolTracker(config, ethrpcClient)
	assert.NoError(t, err)

	// Test parameters
	poolAddress := "0x14cf6d2fe3e1b326114b07d22a6f6bb59e346c67"
	binCounter := uint32(41)

	// Get full pool state
	bins, binPositions, err := tracker.getFullPoolState(context.Background(), poolAddress, binCounter)
	assert.NoError(t, err)
	assert.NotNil(t, bins)
	assert.NotNil(t, binPositions)

	fmt.Println("bins", bins)
	// Log some basic info
	t.Logf("Number of bins: %d", len(bins))
	t.Logf("Number of bin positions: %d", len(binPositions))

	// Verify bin data
	for binId, bin := range bins {
		assert.NotNil(t, bin, "Bin %d should not be nil", binId)
		assert.NotNil(t, bin.TotalSupply, "Bin %d total supply should not be nil", binId)
		assert.NotNil(t, bin.ReserveA, "Bin %d reserveA should not be nil", binId)
		assert.NotNil(t, bin.ReserveB, "Bin %d reserveB should not be nil", binId)
	}
}
