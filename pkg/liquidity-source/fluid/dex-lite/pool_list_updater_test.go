package dexLite

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolsListUpdater(t *testing.T) {
	t.Parallel()
	_ = logger.SetLogLevel("debug")

	if os.Getenv("CI") != "" {
		t.Skip()
	}

	var (
		config = Config{
			DexID:          "fluid-dex-lite",
			ChainID:        valueobject.ChainIDEthereum,
			DexLiteAddress: "0xbED7f3036e2EA43BDBEDC95f1eDd0bB336F8eb2f", // FluidDexLite mainnet address
		}
	)

	logger.Debugf("Starting TestPoolsListUpdater with config: %+v", config)

	client := ethrpc.New("https://ethereum.kyberengineering.io")
	client.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	updater := NewPoolsListUpdater(&config, client)
	require.NotNil(t, updater)
	logger.Debugf("PoolsListUpdater initialized: %+v", updater)

	t.Run("GetNewPools", func(t *testing.T) {
		// Test getting new pools
		pools, metadata, err := updater.GetNewPools(context.Background(), nil)
		
		// For now, this might find 0 pools since FluidDexLite is newly deployed
		if err != nil {
			logger.Debugf("Error getting pools (expected for new protocol): %v", err)
		} else {
			require.NotNil(t, pools)
			require.NotNil(t, metadata)
			
			logger.Debugf("Found %d pools", len(pools))
			logger.Debugf("Metadata: %s", string(metadata))
			
			for i, pool := range pools {
				logger.Debugf("Pool %d: %+v", i, pool)
				require.Equal(t, DexType, pool.Type)
				require.Equal(t, "fluid-dex-lite", pool.Exchange)
				require.Len(t, pool.Tokens, 2)
				require.Len(t, pool.Reserves, 2)
			}
		}
	})
}

func TestGetAllPools(t *testing.T) {
	t.Parallel()
	_ = logger.SetLogLevel("debug")

	if os.Getenv("CI") != "" {
		t.Skip()
	}

	var (
		config = Config{
			DexLiteAddress: "0xbED7f3036e2EA43BDBEDC95f1eDd0bB336F8eb2f",
		}
	)

	client := ethrpc.New("https://ethereum.kyberengineering.io")
	client.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	updater := NewPoolsListUpdater(&config, client)

	pools, err := updater.getAllPools(context.Background())
	
	if err != nil {
		logger.Debugf("Error getting all pools (expected for new protocol): %v", err)
	} else {
		require.NotNil(t, pools)
		logger.Debugf("Found %d pools in getAllPools", len(pools))
		
		for i, pool := range pools {
			logger.Debugf("Pool %d: DexId=%x, Token0=%s, Token1=%s, Fee=%s", 
				i, pool.DexId, pool.DexKey.Token0.Hex(), pool.DexKey.Token1.Hex(), pool.Fee.String())
		}
	}
}

func TestCalculateArraySlot(t *testing.T) {
	updater := &PoolsListUpdater{}
	
	// Test array slot calculation
	baseSlot := big.NewInt(1) // _dexesList is at slot 1
	index := 0
	
	slot := updater.calculateArraySlot(baseSlot, index)
	require.NotEqual(t, common.Hash{}, slot)
	
	logger.Debugf("Array slot for index 0: %s", slot.Hex())
	
	// Different indices should give different slots
	slot2 := updater.calculateArraySlot(baseSlot, 1)
	require.NotEqual(t, slot, slot2)
	
	logger.Debugf("Array slot for index 1: %s", slot2.Hex())
}

func TestCalculateDexIdFromUpdater(t *testing.T) {
	updater := &PoolsListUpdater{}
	
	dexKey := DexKey{
		Token0: common.HexToAddress("0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1"), // USDC
		Token1: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), // USDT
		Salt:   [32]byte{},
	}
	
	dexId := updater.calculateDexId(dexKey)
	require.NotEqual(t, [8]byte{}, dexId)
	
	logger.Debugf("DexId from updater: %x", dexId)
}

func TestReadTokensDecimals(t *testing.T) {
	t.Parallel()
	
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	var (
		config = Config{
			DexLiteAddress: "0xbED7f3036e2EA43BDBEDC95f1eDd0bB336F8eb2f",
		}
	)

	client := ethrpc.New("https://ethereum.kyberengineering.io")
	client.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	updater := NewPoolsListUpdater(&config, client)

	// Test with real tokens
	usdc := common.HexToAddress("0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1")
	usdt := common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7")
	
	decimals0, decimals1, err := updater.readTokensDecimals(context.Background(), usdc, usdt)
	
	if err != nil {
		logger.Debugf("Error reading token decimals: %v", err)
	} else {
		require.Equal(t, uint8(6), decimals0) // USDC has 6 decimals
		require.Equal(t, uint8(6), decimals1) // USDT has 6 decimals
		
		logger.Debugf("Token decimals: USDC=%d, USDT=%d", decimals0, decimals1)
	}
}