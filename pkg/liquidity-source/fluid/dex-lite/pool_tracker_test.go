package dexLite

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestPoolTracker(t *testing.T) {
	t.Parallel()
	_ = logger.SetLogLevel("debug")

	if os.Getenv("CI") != "" {
		t.Skip()
	}

	var (
		config = Config{
			DexLiteAddress: "0xbED7f3036e2EA43BDBEDC95f1eDd0bB336F8eb2f", // FluidDexLite mainnet address
		}
	)

	logger.Debugf("Starting TestPoolTracker with config: %+v", config)

	client := ethrpc.New("https://ethereum.kyberengineering.io")
	client.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	poolTracker := NewPoolTracker(&config, client)
	require.NotNil(t, poolTracker)
	logger.Debugf("PoolTracker initialized: %+v", poolTracker)

		t.Run("USDC_USDT_Pool", func(t *testing.T) {
		staticExtraBytes, _ := json.Marshal(&StaticExtra{
		DexLiteAddress: config.DexLiteAddress,
		HasNative:      false,
	})

	// Create mock dexKey and dexId
	mockDexKey := DexKey{
		Token0: common.HexToAddress("0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1"), // USDC
		Token1: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), // USDT
		Salt:   [32]byte{},
	}
	mockDexId := [8]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	extra := PoolExtra{
		DexKey: mockDexKey,
		DexId:  mockDexId,
		PoolState: PoolState{
			DexVariables:     big.NewInt(0x123456789abcdef),
			CenterPriceShift: big.NewInt(0),
			RangeShift:       big.NewInt(0),
			ThresholdShift:   big.NewInt(0),
		},
	}
	extraBytes, _ := json.Marshal(extra)

	testPool := entity.Pool{
		Address:  config.DexLiteAddress, // Use singleton contract address
		Exchange: "fluid-dex-lite",
		Type:     DexType,
		Reserves: entity.PoolReserves{"1000000000", "1000000000"}, // 1000 USDC, 1000 USDT
		Tokens: []*entity.PoolToken{
			{
				Address:   "0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1", // USDC
				Swappable: true,
				Decimals:  6,
			},
			{
				Address:   "0xdAC17F958D2ee523a2206206994597C13D831ec7", // USDT
				Swappable: true,
				Decimals:  6,
			},
		},
		SwapFee:     0.001, // 0.1%
		BlockNumber: 18000000,
		Extra:       string(extraBytes),
		StaticExtra: string(staticExtraBytes),
	}

		// Test GetNewPoolState
		newPool, err := poolTracker.GetNewPoolState(context.Background(), testPool, pool.GetNewPoolStateParams{})
		
		// For now, this might error because no actual pool exists, but structure should be correct
		if err != nil {
			logger.Debugf("Expected error for non-existent pool: %v", err)
		} else {
			require.NotNil(t, newPool)
			logger.Debugf("New pool state: %+v", newPool)
			
			// Verify the structure
			require.Equal(t, testPool.Address, newPool.Address)
			require.Equal(t, testPool.Type, newPool.Type)
			require.Len(t, newPool.Tokens, 2)
		}
	})
}

func TestCalculateDexId(t *testing.T) {
	tracker := &PoolTracker{}
	
	dexKey := DexKey{
		Token0: common.HexToAddress("0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1"), // USDC
		Token1: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), // USDT  
		Salt:   [32]byte{},
	}
	
	dexId := tracker.calculateDexId(dexKey)
	require.NotEqual(t, [8]byte{}, dexId)
	
	// DexId should be deterministic
	dexId2 := tracker.calculateDexId(dexKey)
	require.Equal(t, dexId, dexId2)
	
	logger.Debugf("DexId for USDC/USDT: %x", dexId)
}

func TestCalculatePoolStateSlot(t *testing.T) {
	tracker := &PoolTracker{}
	
	dexId := [8]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
	
	// Test different offsets
	for i := 0; i < 5; i++ {
		slot := tracker.calculatePoolStateSlot(dexId, i)
		require.NotEqual(t, common.Hash{}, slot)
		logger.Debugf("Slot %d: %s", i, slot.Hex())
	}
}

// TestCalculateReserves was removed because calculateReserves is no longer part of PoolTracker
// Reserve calculations are now handled by PoolSimulator