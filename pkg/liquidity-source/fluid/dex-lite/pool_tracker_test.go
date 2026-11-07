package dexLite

import (
	"context"
	"math/big"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
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

	client := ethrpc.New("https://ethereum-rpc.kyberswap.com")
	client.SetMulticallContract(common.HexToAddress("0x5ba1e12693dc8f9c48aad8770482f4739beed696"))

	poolTracker := NewPoolTracker(&config, client)
	require.NotNil(t, poolTracker)
	logger.Debugf("PoolTracker initialized: %+v", poolTracker)

	t.Run("USDC_USDT_Pool", func(t *testing.T) {
		// Create mock dexKey and dexId
		mockDexKey := DexKey{
			Token0: common.HexToAddress("0xA0b86a33E6441c0c37Fc0C16b6C7Da2A0edD0bD1"), // USDC
			Token1: common.HexToAddress("0xdAC17F958D2ee523a2206206994597C13D831ec7"), // USDT
			Salt:   common.Hash{},
		}
		mockDexId := [8]byte{0x12, 0x34, 0x56, 0x78, 0x9a, 0xbc, 0xde, 0xf0}

		staticExtraBytes, _ := json.Marshal(&StaticExtra{
			DexLiteAddress: config.DexLiteAddress,
			DexKey:         mockDexKey,
			DexId:          mockDexId,
		})
		extra := PoolExtra{
			PoolState: PoolState{
				DexVariables:     uint256.NewInt(0x123456789abcdef),
				CenterPriceShift: uint256.NewInt(0),
				RangeShift:       uint256.NewInt(0),
				NewCenterPrice:   uint256.NewInt(0),
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

	client := ethrpc.New("https://ethereum-rpc.kyberswap.com")
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
	for i := range 5 {
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
		testPoolState, _, err := poolTracker.getPoolStateByDexId(context.Background(), testDexId, testDexKey, nil)
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

	poolState, blockNumber, err := poolTracker.getPoolStateByDexId(ctx, dexId, dexKey, nil)
	if err != nil {
		logger.Debugf("❌ Failed to read pool state directly: %v", err)
	} else {
		logger.Debugf("✅ RAW POOL STATE FROM BLOCKCHAIN:")
		logger.Debugf("   Block Number: %d", blockNumber)
		logger.Debugf("   DexVariables (raw): %s", poolState.DexVariables.String())
		logger.Debugf("   DexVariables (hex): 0x%s", poolState.DexVariables.Hex())
		logger.Debugf("   CenterPriceShift: %s", poolState.CenterPriceShift.String())
		logger.Debugf("   RangeShift: %s", poolState.RangeShift.String())
		logger.Debugf("   NewCenterPrice: %s", poolState.NewCenterPrice.String())

		if poolState.DexVariables.Sign() != 0 {
			logger.Debugf("\n🔍 DECODING REAL DEXVARIABLES FROM ON-CHAIN:")

			// Use pool tracker's unpack method
			unpacked := unpackDexVariables(poolState.DexVariables)
			logger.Debugf("\n📊 DECODED ON-CHAIN DEX VARIABLES:")
			logger.Debugf("   Fee: %s basis points (%.6f%%)", unpacked.Fee.String(), unpacked.Fee.Float64()/10000)
			logger.Debugf("   Revenue Cut: %s", unpacked.RevenueCut.String())
			logger.Debugf("   Rebalancing Status: %d", unpacked.RebalancingStatus)
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

func TestPoolTrackerCenterPrice(t *testing.T) {
	multicallABI, err := abi.JSON(strings.NewReader(`[{"name":"tryBlockAndAggregate","type":"function","inputs":[{"name":"requireSuccess","type":"bool"},{"name":"calls","type":"tuple[]","components":[{"name":"target","type":"address"},{"name":"callData","type":"bytes"}]}],"outputs":[{"name":"blockNumber","type":"uint256"},{"name":"blockHash","type":"bytes32"},{"name":"returnData","type":"tuple[]","components":[{"name":"success","type":"bool"},{"name":"returnData","type":"bytes"}]}]}]`))
	require.NoError(t, err)
	for _, tc := range []struct {
		name          string
		withOverrides bool
		sourceNonce   uint64
		block         uint64
		priceError    bool
	}{
		{name: "external price", sourceNonce: 7, block: 12345},
		{name: "external price with overrides", sourceNonce: 7, block: 12345, withOverrides: true},
		{name: "external price failure preserves the pool", sourceNonce: 7, block: 12345, priceError: true},
		{name: "no external source", block: 12345},
		{name: "unknown block keeps latest", sourceNonce: 7},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := Config{
				DexLiteAddress:  "0xbbcb91440523216e2b87052a99f69c604a7b6e00",
				DeployerAddress: common.HexToAddress("0x4ec7b668baf70d4a4b0fc7941a7708a07b6d45be"),
			}
			static := StaticExtra{
				DexLiteAddress: config.DexLiteAddress,
				DexId:          DexId{1, 2, 3, 4, 5, 6, 7, 8},
				DexKey: DexKey{
					Token0: common.HexToAddress("0xeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeeee"),
					Token1: common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
				},
			}
			staticBytes, err := json.Marshal(static)
			require.NoError(t, err)
			original := entity.Pool{
				Address: "fluid_pool", StaticExtra: string(staticBytes), Extra: "previous state", BlockNumber: 100,
				Tokens: []*entity.PoolToken{
					{Address: "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2", Decimals: 18},
					{Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", Decimals: 6},
				},
				Reserves: entity.PoolReserves{"123", "456"},
			}
			dexVariables := new(big.Int).Lsh(new(big.Int).SetUint64(tc.sourceNonce), BitPosCenterPriceContractAddress)
			if tc.sourceNonce != 0 {
				dexVariables.SetBit(dexVariables, BitPosCenterPriceShiftActive, 1)
			}
			dexVariables.Or(dexVariables, new(big.Int).Lsh(big.NewInt(100), BitPosToken0TotalSupplyAdjusted))
			dexVariables.Or(dexVariables, new(big.Int).Lsh(big.NewInt(200), BitPosToken1TotalSupplyAdjusted))
			slots := []common.Hash{StorageSlotDexVariables, StorageSlotCenterPriceShift, StorageSlotRangeShift}
			values := []*big.Int{dexVariables, big.NewInt(123), big.NewInt(456)}
			price := new(big.Int).Exp(big.NewInt(10), big.NewInt(27), nil)
			multicall := common.HexToAddress("0x0000000000000000000000000000000000000010")
			var overrides map[common.Address]gethclient.OverrideAccount
			if tc.withOverrides {
				overrides = map[common.Address]gethclient.OverrideAccount{
					static.DexKey.Token1: {Code: []byte{0x60, 0x00}},
				}
			}
			var mu sync.Mutex
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var request struct {
					ID     json.RawMessage   `json:"id"`
					Method string            `json:"method"`
					Params []json.RawMessage `json:"params"`
				}
				if !assert.NoError(t, json.NewDecoder(r.Body).Decode(&request)) || !assert.GreaterOrEqual(t, len(request.Params), 2) {
					return
				}
				mu.Lock()
				requestCount++
				n := requestCount
				mu.Unlock()
				assert.Equal(t, "eth_call", request.Method)
				var call struct {
					To    common.Address
					Data  string
					Input string
				}
				if !assert.NoError(t, json.Unmarshal(request.Params[0], &call)) {
					return
				}
				input := call.Data
				if input == "" {
					input = call.Input
				}
				data, err := hexutil.Decode(input)
				if !assert.NoError(t, err) || !assert.GreaterOrEqual(t, len(data), 4) {
					return
				}
				wantTag := "latest"
				if n > 1 && tc.block > 0 {
					wantTag = hexutil.EncodeUint64(tc.block)
				}
				assert.JSONEq(t, `"`+wantTag+`"`, string(request.Params[1]))
				if tc.withOverrides {
					if !assert.Len(t, request.Params, 3) {
						return
					}
					want, err := json.Marshal(overrides)
					if !assert.NoError(t, err) {
						return
					}
					assert.JSONEq(t, string(want), string(request.Params[2]))
				} else {
					assert.Len(t, request.Params, 2)
				}
				var result []byte
				if n == 1 {
					assert.Equal(t, multicall, call.To)
					method := multicallABI.Methods["tryBlockAndAggregate"]
					assert.Equal(t, method.ID, data[:4])
					args, err := method.Inputs.Unpack(data[4:])
					if !assert.NoError(t, err) {
						return
					}
					assert.Equal(t, true, args[0])
					calls := *abi.ConvertType(args[1], new([]struct {
						Target   common.Address
						CallData []byte
					})).(*[]struct {
						Target   common.Address
						CallData []byte
					})
					if !assert.Len(t, calls, 3) {
						return
					}
					outputs := make([]struct {
						Success    bool
						ReturnData []byte
					}, len(calls))
					read := fluidDexLiteABI.Methods[SRMethodReadFromStorage]
					for i, inner := range calls {
						assert.Equal(t, common.HexToAddress(config.DexLiteAddress), inner.Target)
						expected, err := read.Inputs.Pack(calculatePoolStateSlot(static.DexId, slots[i]))
						if !assert.NoError(t, err) {
							return
						}
						assert.Equal(t, append(read.ID, expected...), inner.CallData)
						outputs[i].Success = true
						outputs[i].ReturnData, err = read.Outputs.Pack(values[i])
						if !assert.NoError(t, err) {
							return
						}
					}
					result, err = method.Outputs.Pack(new(big.Int).SetUint64(tc.block), common.Hash{}, outputs)
					if !assert.NoError(t, err) {
						return
					}
				} else {
					assert.Equal(t, 2, n)
					assert.Equal(t, crypto.CreateAddress(config.DeployerAddress, tc.sourceNonce), call.To)
					method := centerPriceABI.Methods[CenterPriceMethodCenterPrice]
					assert.Equal(t, method.ID, data[:4])
					args, err := method.Inputs.Unpack(data[4:])
					if !assert.NoError(t, err) {
						return
					}
					assert.Equal(t, []any{static.DexKey.Token0, static.DexKey.Token1}, args)
					if tc.priceError {
						assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "error": map[string]any{"code": -32000, "message": "execution reverted: price unavailable"}}))
						return
					}
					result, err = method.Outputs.Pack(price)
					if !assert.NoError(t, err) {
						return
					}
				}
				assert.NoError(t, json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": hexutil.Encode(result)}))
			}))
			defer server.Close()
			tracker := NewPoolTracker(&config, ethrpc.New(server.URL).SetMulticallContract(multicall))
			var updated entity.Pool
			if tc.withOverrides {
				updated, err = tracker.GetNewPoolStateWithOverrides(context.Background(), original, pool.GetNewPoolStateWithOverridesParams{Overrides: overrides})
			} else {
				updated, err = tracker.GetNewPoolState(context.Background(), original, pool.GetNewPoolStateParams{})
			}
			if tc.priceError {
				require.ErrorContains(t, err, "price unavailable")
				require.Equal(t, original, updated)
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.block, updated.BlockNumber)
				var extra PoolExtra
				require.NoError(t, json.Unmarshal([]byte(updated.Extra), &extra))
				require.Equal(t, uint256.MustFromBig(dexVariables), extra.PoolState.DexVariables)
				require.Equal(t, uint256.NewInt(123), extra.PoolState.CenterPriceShift)
				require.Equal(t, uint256.NewInt(456), extra.PoolState.RangeShift)
				require.Equal(t, entity.PoolReserves{"100000000000", "0"}, updated.Reserves)
				if tc.sourceNonce == 0 {
					require.True(t, extra.PoolState.NewCenterPrice.IsZero())
				} else {
					require.Equal(t, uint256.MustFromBig(price), extra.PoolState.NewCenterPrice)
				}
			}
			mu.Lock()
			defer mu.Unlock()
			wantCount := 2
			if tc.sourceNonce == 0 {
				wantCount = 1
			}
			require.Equal(t, wantCount, requestCount)
		})
	}
}
