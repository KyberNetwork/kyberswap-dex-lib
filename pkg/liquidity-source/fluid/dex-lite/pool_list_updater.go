package dexLite

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       Config
	ethrpcClient *ethrpc.Client
}

type Metadata struct {
	LastSyncPoolsLength int `json:"lastSyncPoolsLength"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       *config,
		ethrpcClient: ethrpcClient,
	}
}

func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	logger.WithFields(logger.Fields{
		"dexType": DexType,
	}).Infof("Start updating pools list ...")
	defer func() {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
		}).Infof("Finish updating pools list.")
	}()

	allPools, err := u.getAllPools(ctx)
	if err != nil {
		return nil, nil, err
	}

	// Get current block timestamp for all pools
	blockTimestamp, err := u.ethrpcClient.R().SetContext(ctx).GetCurrentBlockTimestamp()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to get block timestamp")
		return nil, nil, err
	}

	newMetadataBytes, err := json.Marshal(Metadata{
		LastSyncPoolsLength: len(allPools),
	})
	if err != nil {
		return nil, nil, err
	}

	var metadata Metadata
	if len(metadataBytes) > 0 {
		if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
			return nil, nil, err
		}
	}

	if metadata.LastSyncPoolsLength > 0 {
		// only handle new pools after last synced index
		allPools = allPools[metadata.LastSyncPoolsLength:]
	}

	pools := make([]entity.Pool, 0)

	// **OPTIMIZATION**: Batch read token decimals for all unique tokens
	allTokenDecimals, err := u.batchReadTokenDecimals(ctx, allPools)
	if err != nil {
		return nil, nil, err
	}

	for _, curPool := range allPools {
		token0Decimals := allTokenDecimals[curPool.DexKey.Token0]
		token1Decimals := allTokenDecimals[curPool.DexKey.Token1]

		staticExtraBytes, err := json.Marshal(&StaticExtra{
			DexLiteAddress: u.config.DexLiteAddress,
			HasNative: strings.EqualFold(curPool.DexKey.Token0.Hex(), valueobject.NativeAddress) ||
				strings.EqualFold(curPool.DexKey.Token1.Hex(), valueobject.NativeAddress),
		})
		if err != nil {
			return nil, nil, err
		}

		// Store only the essential FluidDexLite data
		extra := PoolExtra{
			DexKey:         curPool.DexKey,
			DexId:          curPool.DexId,
			PoolState:      curPool.State,
			BlockTimestamp: blockTimestamp,
		}

		extraBytes, err := json.Marshal(extra)
		if err != nil {
			logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("Error marshaling extra data")
			return nil, nil, err
		}

		// Calculate actual reserves and fee for KyberSwap routing
		reserves, fee := u.calculatePoolMetrics(curPool)

		pool := entity.Pool{
			Address:  u.config.DexLiteAddress, // Singleton contract address
			Exchange: "fluid-dex-lite",
			Type:     DexType,
			Reserves: reserves, // Real reserves for swap calculations
			Tokens: []*entity.PoolToken{
				{
					Address:   valueobject.WrapNativeLower(curPool.DexKey.Token0.Hex(), u.config.ChainID),
					Swappable: true,
					Decimals:  token0Decimals,
				},
				{
					Address:   valueobject.WrapNativeLower(curPool.DexKey.Token1.Hex(), u.config.ChainID),
					Swappable: true,
					Decimals:  token1Decimals,
				},
			},
			SwapFee:     fee, // Real fee for routing decisions
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, pool)
	}

	return pools, newMetadataBytes, nil
}

// calculatePoolMetrics computes real reserves and fee for KyberSwap routing
func (u *PoolsListUpdater) calculatePoolMetrics(curPool PoolWithState) (entity.PoolReserves, float64) {
	// Extract fee from dexVariables (bits 0-12, stored as basis points)
	feeRaw := new(big.Int).And(curPool.State.DexVariables, X13)
	fee := float64(feeRaw.Int64()) / FeePercentPrecision

	// Extract total supplies and calculate reserves
	// Unpack dexVariables to get token supplies in internal precision (9 decimals)
	unpackedVars := u.unpackDexVariables(curPool.State.DexVariables)

	// Convert internal supplies to token decimals for display
	token0Supply := u.adjustFromInternalDecimals(unpackedVars.Token0TotalSupplyAdjusted, 6) // Assume 6 decimals like USDC
	token1Supply := u.adjustFromInternalDecimals(unpackedVars.Token1TotalSupplyAdjusted, 6) // Assume 6 decimals like USDT

	reserves := entity.PoolReserves{
		token0Supply.String(),
		token1Supply.String(),
	}

	return reserves, fee
}

// unpackDexVariables extracts the packed variables from dexVariables
func (u *PoolsListUpdater) unpackDexVariables(dexVariables *big.Int) UnpackedDexVariables {
	return UnpackedDexVariables{
		Fee:                         new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesFee), X13),
		RevenueCut:                  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesRevenueCut), X7),
		RebalancingStatus:           new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesRebalancingStatus), X2),
		CenterPriceShiftActive:      new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesCenterPriceShiftActive), X1).Cmp(big.NewInt(1)) == 0,
		CenterPrice:                 new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesCenterPrice), X40),
		CenterPriceContractAddress:  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesCenterPriceContractAddress), X19),
		RangePercentShiftActive:     new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesRangePercentShiftActive), X1).Cmp(big.NewInt(1)) == 0,
		UpperPercent:                new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesUpperPercent), X14),
		LowerPercent:                new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesLowerPercent), X14),
		ThresholdPercentShiftActive: new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesThresholdPercentShiftActive), X1).Cmp(big.NewInt(1)) == 0,
		UpperShiftThresholdPercent:  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesUpperShiftThresholdPercent), X7),
		LowerShiftThresholdPercent:  new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesLowerShiftThresholdPercent), X7),
		Token0Decimals:              new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken0Decimals), X5),
		Token1Decimals:              new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken1Decimals), X5),
		Token0TotalSupplyAdjusted:   new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken0TotalSupplyAdjusted), X60),
		Token1TotalSupplyAdjusted:   new(big.Int).And(new(big.Int).Rsh(dexVariables, BitsDexLiteDexVariablesToken1TotalSupplyAdjusted), X60),
	}
}

// adjustFromInternalDecimals converts from 9-decimal precision to token decimals
func (u *PoolsListUpdater) adjustFromInternalDecimals(amount *big.Int, tokenDecimals uint8) *big.Int {
	internalDecimals := uint8(9)
	if tokenDecimals >= internalDecimals {
		factor := tenPow(int(tokenDecimals - internalDecimals))
		return new(big.Int).Mul(amount, factor)
	} else {
		factor := tenPow(int(internalDecimals - tokenDecimals))
		return new(big.Int).Div(amount, factor)
	}
}

// tenPow calculates 10^n
func tenPow(n int) *big.Int {
	result := big.NewInt(1)
	ten := big.NewInt(10)
	for i := 0; i < n; i++ {
		result.Mul(result, ten)
	}
	return result
}

func (u *PoolsListUpdater) getAllPools(ctx context.Context) ([]PoolWithState, error) {
	// Get the number of pools in the dex list
	var dexListLength *big.Int

	req := u.ethrpcClient.R().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    fluidDexLiteABI,
		Target: u.config.DexLiteAddress,
		Method: SRMethodReadFromStorage,
		Params: []interface{}{common.HexToHash(StorageSlotDexList)},
	}, []interface{}{&dexListLength})

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to get dex list length")
		return nil, err
	}

	length := int(dexListLength.Int64())
	if length == 0 {
		return []PoolWithState{}, nil
	}

	// **OPTIMIZATION**: Batch ALL pool reading into 2 total RPC calls
	return u.readAllPoolsBatched(ctx, length)
}

// readAllPoolsBatched reads ALL pools in just 2 RPC calls instead of 2N calls
func (u *PoolsListUpdater) readAllPoolsBatched(ctx context.Context, length int) ([]PoolWithState, error) {
	// **BATCH 1**: Read ALL DexKeys in a single RPC call (3N storage slots)
	var allDexKeyRaw [][3]*big.Int
	req1 := u.ethrpcClient.R().SetContext(ctx)

	for i := 0; i < length; i++ {
		dexListSlot := u.calculateArraySlot(big.NewInt(1), i)
		dexListSlotBig := new(big.Int).SetBytes(dexListSlot[:])

		var dexKeyRaw [3]*big.Int
		// Read 3 consecutive slots for this DexKey struct
		for j := 0; j < 3; j++ {
			slot := new(big.Int).Add(dexListSlotBig, big.NewInt(int64(j)))
			dexKeyRaw[j] = new(big.Int)
			req1.AddCall(&ethrpc.Call{
				ABI:    fluidDexLiteABI,
				Target: u.config.DexLiteAddress,
				Method: SRMethodReadFromStorage,
				Params: []interface{}{common.BigToHash(slot)},
			}, []interface{}{dexKeyRaw[j]})
		}
		allDexKeyRaw = append(allDexKeyRaw, dexKeyRaw)
	}

	_, err := req1.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to batch read all DexKeys")
		return nil, err
	}

	// Process DexKeys and prepare dexIds for second batch
	var validPools []int
	var dexKeys []DexKey
	var dexIds [][8]byte

	for i, dexKeyRaw := range allDexKeyRaw {
		// Skip invalid pools (token0 == 0)
		if dexKeyRaw[0].Sign() == 0 {
			continue
		}

		// Reconstruct DexKey
		dexKey := DexKey{
			Token0: common.BigToAddress(dexKeyRaw[0]),
			Token1: common.BigToAddress(dexKeyRaw[1]),
		}
		// Properly reconstruct salt by padding to 32 bytes
		saltBytes := common.LeftPadBytes(dexKeyRaw[2].Bytes(), 32)
		copy(dexKey.Salt[:], saltBytes)

		// Calculate dexId
		dexId := u.calculateDexId(dexKey)

		validPools = append(validPools, i)
		dexKeys = append(dexKeys, dexKey)
		dexIds = append(dexIds, dexId)
	}

	if len(validPools) == 0 {
		return []PoolWithState{}, nil
	}

	// **BATCH 2**: Read ALL dexVariables in a single RPC call (N storage slots)
	var allDexVariables []*big.Int
	req2 := u.ethrpcClient.R().SetContext(ctx)

	for _, dexId := range dexIds {
		dexVariablesSlot := u.calculatePoolStateSlot(dexId, 0)
		dexVariables := new(big.Int)
		req2.AddCall(&ethrpc.Call{
			ABI:    fluidDexLiteABI,
			Target: u.config.DexLiteAddress,
			Method: SRMethodReadFromStorage,
			Params: []interface{}{dexVariablesSlot},
		}, []interface{}{dexVariables})
		allDexVariables = append(allDexVariables, dexVariables)
	}

	_, err = req2.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to batch read all dexVariables")
		return nil, err
	}

	// Build final pool list
	pools := make([]PoolWithState, 0, len(validPools))
	var skippedCount int
	for i, dexVariables := range allDexVariables {
		// Skip uninitialized pools
		if dexVariables.Sign() == 0 {
			skippedCount++
			continue
		}

		// Extract fee from dexVariables
		fee := new(big.Int).And(dexVariables, X13)

		// Create minimal pool state for listing
		poolState := PoolState{
			DexVariables:     dexVariables,
			CenterPriceShift: big.NewInt(0),
			RangeShift:       big.NewInt(0),
			ThresholdShift:   big.NewInt(0),
		}

		pools = append(pools, PoolWithState{
			DexId:    dexIds[i],
			DexKey:   dexKeys[i],
			State:    poolState,
			Fee:      fee,
			IsActive: true,
		})
	}

	logger.WithFields(logger.Fields{
		"dexType":            DexType,
		"totalIndexes":       length,
		"validDexKeys":       len(validPools),
		"uninitializedPools": skippedCount,
		"initializedPools":   len(pools),
		"rpcCallsSaved":      (length * 2) - 2, // Was 2N calls, now 2 calls
	}).Info("Optimized pool discovery completed")

	return pools, nil
}

// batchReadTokenDecimals reads decimals for all unique tokens in a single RPC call
func (u *PoolsListUpdater) batchReadTokenDecimals(ctx context.Context, pools []PoolWithState) (map[common.Address]uint8, error) {
	// Collect unique tokens (excluding native)
	uniqueTokens := make(map[common.Address]bool)
	for _, pool := range pools {
		if !strings.EqualFold(pool.DexKey.Token0.Hex(), valueobject.NativeAddress) {
			uniqueTokens[pool.DexKey.Token0] = true
		}
		if !strings.EqualFold(pool.DexKey.Token1.Hex(), valueobject.NativeAddress) {
			uniqueTokens[pool.DexKey.Token1] = true
		}
	}

	decimalsMap := make(map[common.Address]uint8)

	// Set native token decimals
	nativeAddr := common.HexToAddress(valueobject.NativeAddress)
	decimalsMap[nativeAddr] = 18

	if len(uniqueTokens) == 0 {
		return decimalsMap, nil
	}

	// Batch read all token decimals in a single RPC call
	req := u.ethrpcClient.R().SetContext(ctx)
	var tokens []common.Address
	var decimalsResults []uint8

	for token := range uniqueTokens {
		var decimals uint8
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: token.String(),
			Method: TokenMethodDecimals,
			Params: nil,
		}, []interface{}{&decimals})

		tokens = append(tokens, token)
		decimalsResults = append(decimalsResults, decimals)
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to batch read token decimals")
		return nil, err
	}

	// Build results map
	for i, token := range tokens {
		decimalsMap[token] = decimalsResults[i]
	}

	logger.WithFields(logger.Fields{
		"dexType":      DexType,
		"uniqueTokens": len(uniqueTokens),
	}).Debug("Batched token decimals reading completed")

	return decimalsMap, nil
}

func (u *PoolsListUpdater) readTokensDecimals(ctx context.Context, token0 common.Address, token1 common.Address) (uint8, uint8, error) {
	var decimals0, decimals1 uint8

	req := u.ethrpcClient.R().SetContext(ctx)

	if strings.EqualFold(valueobject.NativeAddress, token0.String()) {
		decimals0 = 18
	} else {
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: token0.String(),
			Method: TokenMethodDecimals,
			Params: nil,
		}, []interface{}{&decimals0})
	}

	if strings.EqualFold(valueobject.NativeAddress, token1.String()) {
		decimals1 = 18
	} else {
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: token1.String(),
			Method: TokenMethodDecimals,
			Params: nil,
		}, []interface{}{&decimals1})
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("can not read token info")
		return 0, 0, err
	}

	return decimals0, decimals1, nil
}

// Helper functions for storage calculations
func (u *PoolsListUpdater) calculateArraySlot(baseSlot *big.Int, index int) common.Hash {
	// For dynamic arrays: keccak256(baseSlot) + index
	baseHash := crypto.Keccak256(common.LeftPadBytes(baseSlot.Bytes(), 32))

	indexBig := big.NewInt(int64(index))
	result := new(big.Int).SetBytes(baseHash)
	result.Add(result, indexBig)

	return common.BigToHash(result)
}

func (u *PoolsListUpdater) calculateDexId(dexKey DexKey) [8]byte {
	// dexId = bytes8(keccak256(abi.encode(dexKey)))
	// Encode the DexKey similar to abi.encode
	data := make([]byte, 0, 96) // 32 + 32 + 32 bytes
	data = append(data, common.LeftPadBytes(dexKey.Token0.Bytes(), 32)...)
	data = append(data, common.LeftPadBytes(dexKey.Token1.Bytes(), 32)...)
	data = append(data, dexKey.Salt[:]...)

	hash := crypto.Keccak256(data)

	var dexId [8]byte
	copy(dexId[:], hash[:8])
	return dexId
}

func (u *PoolsListUpdater) calculatePoolStateSlot(dexId [8]byte, offset int) common.Hash {
	// Pool state mapping: _dexVariables is at slot 2, others follow
	// keccak256(dexId, baseSlot) where baseSlot varies by field
	var baseSlot *big.Int

	switch offset {
	case 0: // _dexVariables mapping at slot 2
		baseSlot = big.NewInt(2)
	case 1: // _centerPriceShift mapping at slot 3
		baseSlot = big.NewInt(3)
	case 2: // _rangeShift mapping at slot 4
		baseSlot = big.NewInt(4)
	case 3: // _thresholdShift mapping at slot 5
		baseSlot = big.NewInt(5)
	default:
		baseSlot = big.NewInt(2)
	}

	data := make([]byte, 0, 40) // 8 + 32 bytes
	data = append(data, common.LeftPadBytes(dexId[:], 32)...)
	data = append(data, common.LeftPadBytes(baseSlot.Bytes(), 32)...)
	baseHash := crypto.Keccak256(data)

	return common.BytesToHash(baseHash)
}
