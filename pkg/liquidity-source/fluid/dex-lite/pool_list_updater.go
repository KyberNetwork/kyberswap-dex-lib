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

	for _, curPool := range allPools {
		token0Decimals, token1Decimals, err := u.readTokensDecimals(ctx, curPool.DexKey.Token0, curPool.DexKey.Token1)
		if err != nil {
			return nil, nil, err
		}

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
		Fee:                       new(big.Int).And(dexVariables, X13),
		RevenueCut:                new(big.Int).And(new(big.Int).Rsh(dexVariables, 13), X7),
		Token0TotalSupplyAdjusted: new(big.Int).And(new(big.Int).Rsh(dexVariables, 100), X64),
		Token1TotalSupplyAdjusted: new(big.Int).And(new(big.Int).Rsh(dexVariables, 164), X64),
		RebalancingStatus:         new(big.Int).And(new(big.Int).Rsh(dexVariables, 228), X28),
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

	pools := make([]PoolWithState, 0, length)

	// Read each pool from the dex list array
	for i := 0; i < length; i++ {
		pool, err := u.readPoolAtIndex(ctx, i)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexType": DexType,
				"index":   i,
				"error":   err,
			}).Error("Failed to read pool at index")
			continue
		}
		if pool != nil {
			pools = append(pools, *pool)
		}
	}

	return pools, nil
}

func (u *PoolsListUpdater) readPoolAtIndex(ctx context.Context, index int) (*PoolWithState, error) {
	// Calculate storage slot for dex list array element
	// Array slot = keccak256(1) + index (slot 1 is _dexesList)
	dexListSlot := u.calculateArraySlot(big.NewInt(1), index)
	dexListSlotBig := new(big.Int).SetBytes(dexListSlot[:])

	var dexKeyRaw [3]*big.Int // DexKey has 3 fields: token0, token1, salt

	req := u.ethrpcClient.R().SetContext(ctx)

	// Read dexKey from dex list (3 consecutive slots for struct)
	for i := 0; i < 3; i++ {
		slot := new(big.Int).Add(dexListSlotBig, big.NewInt(int64(i)))
		dexKeyRaw[i] = new(big.Int)
		req.AddCall(&ethrpc.Call{
			ABI:    fluidDexLiteABI,
			Target: u.config.DexLiteAddress,
			Method: SRMethodReadFromStorage,
			Params: []interface{}{common.BigToHash(slot)},
		}, []interface{}{dexKeyRaw[i]})
	}

	_, err := req.Aggregate()
	if err != nil {
		return nil, err
	}

	// Check if dexKey is valid (token0 != 0)
	if dexKeyRaw[0].Sign() == 0 {
		return nil, nil // Empty slot
	}

	// Reconstruct DexKey
	dexKey := DexKey{
		Token0: common.BigToAddress(dexKeyRaw[0]),
		Token1: common.BigToAddress(dexKeyRaw[1]),
	}
	copy(dexKey.Salt[:], dexKeyRaw[2].Bytes())

	// Calculate dexId from dexKey
	dexId := u.calculateDexId(dexKey)

	// Read just dexVariables to get fee for pool listing
	dexVariablesSlot := u.calculatePoolStateSlot(dexId, 0)
	var dexVariables *big.Int = new(big.Int)
	req2 := u.ethrpcClient.R().SetContext(ctx)
	req2.AddCall(&ethrpc.Call{
		ABI:    fluidDexLiteABI,
		Target: u.config.DexLiteAddress,
		Method: SRMethodReadFromStorage,
		Params: []interface{}{dexVariablesSlot},
	}, []interface{}{dexVariables})

	_, err = req2.Aggregate()
	if err != nil {
		return nil, err
	}

	// Check if pool is initialized (has non-zero dexVariables)
	if dexVariables.Sign() == 0 {
		return nil, nil // Pool not initialized
	}

	// Extract fee from dexVariables
	fee := new(big.Int).And(dexVariables, X13)

	// Create minimal pool state for listing - real state will be fetched by tracker
	poolState := PoolState{
		DexVariables:     dexVariables,
		CenterPriceShift: big.NewInt(0),
		RangeShift:       big.NewInt(0),
		ThresholdShift:   big.NewInt(0),
	}

	return &PoolWithState{
		DexId:    dexId,
		DexKey:   dexKey,
		State:    poolState,
		Fee:      fee,
		IsActive: true,
	}, nil
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
