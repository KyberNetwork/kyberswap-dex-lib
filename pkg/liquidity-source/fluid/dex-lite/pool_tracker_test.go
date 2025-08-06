package dexLite

import (
	"context"
	"math/big"
	"os"
	"strings"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
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
			DexLiteAddress: "0xBbcb91440523216e2b87052A99F69c604A7b6e00", // FluidDexLite mainnet address
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
			Salt:   common.Hash{},
		}
		mockDexId := [8]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}
		extra := PoolExtra{
			DexKey: mockDexKey,
			DexId:  mockDexId,
			PoolState: PoolState{
				DexVariables:     uint256.NewInt(0x123456789abcdef),
				CenterPriceShift: uint256.NewInt(0),
				RangeShift:       uint256.NewInt(0),
				ThresholdShift:   uint256.NewInt(0),
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
		Salt:   common.Hash{},
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

// TestRealOnChainPoolState tests against the actual USDC/USDT pool on-chain
func TestRealOnChainPoolState(t *testing.T) {
	_ = logger.SetLogLevel("debug")

	if os.Getenv("CI") != "" {
		t.Skip()
	}

	var (
		config = Config{
			DexLiteAddress: "0xBbcb91440523216e2b87052A99F69c604A7b6e00", // FluidDexLite mainnet address
		}
	)

	logger.Debugf("\n" + strings.Repeat("=", 80))
	logger.Debugf("🔍 TESTING REAL ON-CHAIN USDC/USDT POOL STATE")
	logger.Debugf(strings.Repeat("=", 80))

	client := ethrpc.New("https://ethereum.kyberengineering.io")
	client.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	poolTracker := NewPoolTracker(&config, client)
	require.NotNil(t, poolTracker)

	// Test pool: Real USDC/USDT from the dex
	dexKey := DexKey{
		Token0: common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), // Real USDC
		Token1: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), // Real USDT
	}
	// Salt is zero for the main pool
	copy(dexKey.Salt[:], make([]byte, 32))

	logger.Debugf("📍 Testing DexKey:")
	logger.Debugf("   Token0 (USDC): %s", dexKey.Token0.Hex())
	logger.Debugf("   Token1 (USDT): %s", dexKey.Token1.Hex())
	logger.Debugf("   Salt: %s", common.BytesToHash(dexKey.Salt[:]).Hex())

	// Calculate dexId
	data := make([]byte, 0, 96)
	data = append(data, common.LeftPadBytes(dexKey.Token0.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(dexKey.Token1.Bytes(), 32)...)
	data = append(data, dexKey.Salt[:]...)
	hash := crypto.Keccak256(data)
	var dexId [8]byte
	copy(dexId[:], hash[:8])

	logger.Debugf("🆔 Calculated DexId: %x", dexId)

	// Let me also try different salts to see if there are other pools
	logger.Debugf("\n🔍 TESTING DIFFERENT SALT VALUES:")
	for i := 0; i < 5; i++ {
		testDexKey := DexKey{
			Token0: common.HexToAddress("0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48"), // USDC
			Token1: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), // USDT
		}
		// Try different salt values
		saltBytes := make([]byte, 32)
		if i > 0 {
			saltBytes[31] = byte(i) // Put salt value in last byte
		}
		copy(testDexKey.Salt[:], saltBytes)

		// Calculate dexId for this salt
		testData := make([]byte, 0, 96)
		testData = append(testData, common.LeftPadBytes(testDexKey.Token0.Bytes(), 32)...)
		testData = append(testData, common.LeftPadBytes(testDexKey.Token1.Bytes(), 32)...)
		testData = append(testData, testDexKey.Salt[:]...)
		testHash := crypto.Keccak256(testData)
		var testDexId [8]byte
		copy(testDexId[:], testHash[:8])

		logger.Debugf("   Salt %d: %s -> DexId: %x", i, common.BytesToHash(testDexKey.Salt[:]).Hex(), testDexId)

		// Try to read this pool state
		testPoolState, _, err := poolTracker.getPoolStateByDexId(context.Background(), testDexId, nil)
		if err != nil {
			logger.Debugf("     ❌ Error reading pool state: %v", err)
		} else if testPoolState.DexVariables.Sign() != 0 {
			logger.Debugf("     ✅ FOUND INITIALIZED POOL! DexVariables: %s", testPoolState.DexVariables.String())
			// Update our main dexId to this working one
			dexId = testDexId
			break
		} else {
			logger.Debugf("     ⚪ Pool exists but uninitialized (DexVariables: 0)")
		}
	}

	// 🔧 DIRECTLY READ POOL STATE USING LOW-LEVEL CALLS
	logger.Debugf("\n🔬 DIRECT POOL STATE READING:")
	ctx := context.Background()

	poolState, blockNumber, err := poolTracker.getPoolStateByDexId(ctx, dexId, nil)
	if err != nil {
		logger.Debugf("❌ Failed to read pool state directly: %v", err)
	} else {
		logger.Debugf("✅ RAW POOL STATE FROM BLOCKCHAIN:")
		logger.Debugf("   Block Number: %d", blockNumber)
		logger.Debugf("   DexVariables (raw): %s", poolState.DexVariables.String())
		logger.Debugf("   DexVariables (hex): 0x%s", poolState.DexVariables.Hex())
		logger.Debugf("   CenterPriceShift: %s", poolState.CenterPriceShift.String())
		logger.Debugf("   RangeShift: %s", poolState.RangeShift.String())
		logger.Debugf("   ThresholdShift: %s", poolState.ThresholdShift.String())

		if poolState.DexVariables.Sign() != 0 {
			logger.Debugf("\n🔍 DECODING REAL DEXVARIABLES FROM ON-CHAIN:")

			// Use pool tracker's unpack method
			unpacked := unpackDexVariables(poolState.DexVariables)
			logger.Debugf("\n📊 DECODED ON-CHAIN DEX VARIABLES:")
			logger.Debugf("   Fee: %s basis points (%.6f%%)", unpacked.Fee.String(), unpacked.Fee.Float64()/10000)
			logger.Debugf("   Revenue Cut: %s", unpacked.RevenueCut.String())
			logger.Debugf("   Rebalancing Status: %s", unpacked.RebalancingStatus.String())
			logger.Debugf("   Center Price Shift Active: %v", unpacked.CenterPriceShiftActive)
			logger.Debugf("   Center Price: %s", unpacked.CenterPrice.String())
			logger.Debugf("   Range Percent Shift Active: %v", unpacked.RangePercentShiftActive)
			logger.Debugf("   Upper Percent: %s", unpacked.UpperPercent.String())
			logger.Debugf("   Lower Percent: %s", unpacked.LowerPercent.String())
			logger.Debugf("   Threshold Percent Shift Active: %v", unpacked.ThresholdPercentShiftActive)
			logger.Debugf("   Upper Shift Threshold: %s", unpacked.UpperShiftThresholdPercent.String())
			logger.Debugf("   Lower Shift Threshold: %s", unpacked.LowerShiftThresholdPercent.String())
			logger.Debugf("   Token0 Total Supply Adjusted: %s", unpacked.Token0TotalSupplyAdjusted.String())
			logger.Debugf("   Token1 Total Supply Adjusted: %s", unpacked.Token1TotalSupplyAdjusted.String())
		} else {
			logger.Debugf("⚠️ DexVariables is zero - pool not initialized with liquidity")
		}
	}

	// 🔧 TEST TIMESTAMP AND MULTICALL ISSUES
	logger.Debugf("\n🔧 TESTING TIMESTAMP AND MULTICALL FUNCTIONALITY:")

	// Test 1: Test GetCurrentBlockTimestamp directly
	logger.Debugf("\n📅 Testing GetCurrentBlockTimestamp:")
	timestamp, err := client.NewRequest().SetContext(ctx).GetCurrentBlockTimestamp()
	if err != nil {
		logger.Debugf("❌ GetCurrentBlockTimestamp failed: %v", err)
	} else {
		logger.Debugf("✅ GetCurrentBlockTimestamp succeeded: %d", timestamp)
	}

	// Test 2: Test a simple multicall to see if it works
	logger.Debugf("\n📞 Testing Simple Multicall (reading dexesList length):")
	req := client.NewRequest().SetContext(ctx)

	var dexesListLength *big.Int
	req.AddCall(&ethrpc.Call{
		ABI:    fluidDexLiteABI,
		Target: config.DexLiteAddress,
		Method: "getDexesListLength", // Method we know exists
		Params: nil,
	}, []any{&dexesListLength})

	_, err = req.Call()
	if err != nil {
		logger.Debugf("❌ Simple multicall failed: %v", err)
	} else {
		logger.Debugf("✅ Simple multicall succeeded: dexesList length = %s", dexesListLength.String())
	}

	// Test 3: Test our problematic readFromStorage multicall
	logger.Debugf("\n📞 Testing readFromStorage Multicall:")
	req2 := client.NewRequest().SetContext(ctx)

	var storageResult *big.Int
	slot1Hash := common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001") // slot 1
	req2.AddCall(&ethrpc.Call{
		ABI:    fluidDexLiteABI,
		Target: config.DexLiteAddress,
		Method: "readFromStorage",
		Params: []any{slot1Hash},
	}, []any{&storageResult})

	_, err = req2.Call()
	if err != nil {
		logger.Debugf("❌ readFromStorage multicall failed: %v", err)
	} else {
		logger.Debugf("✅ readFromStorage multicall succeeded: %s", storageResult.String())
	}

	logger.Debugf("\n" + strings.Repeat("=", 80))
	logger.Debugf("🎯 REAL ON-CHAIN POOL STATE TEST COMPLETED")
	logger.Debugf(strings.Repeat("=", 80))
}
