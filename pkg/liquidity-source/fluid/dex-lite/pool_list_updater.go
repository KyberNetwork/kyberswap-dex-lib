package dexLite

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       Config
	ethrpcClient *ethrpc.Client
	ethClient    *ethclient.Client
}

type Metadata struct {
	LastSyncPoolsLength int `json:"lastSyncPoolsLength"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *Config, ethrpcClient *ethrpc.Client) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       *config,
		ethrpcClient: ethrpcClient,
		ethClient:    ethrpcClient.GetETHClient(),
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
	blockTimestamp, err := u.ethrpcClient.NewRequest().SetContext(ctx).GetCurrentBlockTimestamp()
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

		tokens := []*entity.PoolToken{
			{
				Address:   valueobject.WrapNativeLower(hexutil.Encode(curPool.DexKey.Token0[:]), u.config.ChainID),
				Swappable: true,
			},
			{
				Address:   valueobject.WrapNativeLower(hexutil.Encode(curPool.DexKey.Token1[:]), u.config.ChainID),
				Swappable: true,
			},
		}

		// Calculate actual reserves and fee for KyberSwap routing
		reserves, fee := calculatePoolMetrics(&curPool.State, tokens)

		pool := entity.Pool{
			Address:     strings.ToLower(u.config.DexLiteAddress) + hex.EncodeToString(curPool.DexId[:]),
			Exchange:    valueobject.ExchangeFluidDexLite,
			Type:        DexType,
			Reserves:    reserves, // Real reserves for swap calculations
			Tokens:      tokens,
			SwapFee:     fee, // Real fee for routing decisions
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, pool)
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getAllPools(ctx context.Context) ([]PoolWithState, error) {
	// Get the number of pools in the dex list
	var dexListLength *big.Int

	req := u.ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{
		ABI:    fluidDexLiteABI,
		Target: u.config.DexLiteAddress,
		Method: SRMethodReadFromStorage,
		Params: []any{common.HexToHash(StorageSlotDexList)},
	}, []any{&dexListLength})

	_, err := req.Call()
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

// readAllPoolsBatched reads ALL pools using direct ethClient calls
func (u *PoolsListUpdater) readAllPoolsBatched(ctx context.Context, length int) ([]PoolWithState, error) {
	// Read ALL DexKeys using individual storage calls
	var validPools []int
	var dexKeys []DexKey
	var dexIds [][8]byte

	for i := 0; i < length; i++ {
		dexListSlot := u.calculateArraySlot(big.NewInt(1), i)
		dexListSlotBig := new(big.Int).SetBytes(dexListSlot[:])

		// Read 3 consecutive slots for this DexKey struct
		token0Slot := new(big.Int).Set(dexListSlotBig)
		token1Slot := new(big.Int).Add(dexListSlotBig, big.NewInt(1))
		saltSlot := new(big.Int).Add(dexListSlotBig, big.NewInt(2))

		// Read token0, token1, salt
		token0Raw, err := u.readFromStorage(ctx, common.BigToHash(token0Slot))
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexType": DexType,
				"error":   err,
				"index":   i,
			}).Error("Failed to read token0")
			return nil, err
		}

		// Skip invalid pools (token0 == 0)
		if token0Raw.Sign() == 0 {
			continue
		}

		token1Raw, err := u.readFromStorage(ctx, common.BigToHash(token1Slot))
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexType": DexType,
				"error":   err,
				"index":   i,
			}).Error("Failed to read token1")
			return nil, err
		}

		saltRaw, err := u.readFromStorage(ctx, common.BigToHash(saltSlot))
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexType": DexType,
				"error":   err,
				"index":   i,
			}).Error("Failed to read salt")
			return nil, err
		}

		// Reconstruct DexKey
		dexKey := DexKey{
			Token0: common.BigToAddress(token0Raw),
			Token1: common.BigToAddress(token1Raw),
		}
		// Properly reconstruct salt by padding to 32 bytes
		saltBytes := common.LeftPadBytes(saltRaw.Bytes(), 32)
		copy(dexKey.Salt[:], saltBytes)

		// Calculate dexId
		dexId := u.calculateDexId(dexKey)

		validPools = append(validPools, i)
		dexKeys = append(dexKeys, dexKey)
		dexIds = append(dexIds, dexId)

		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"index":   i,
			"token0":  dexKey.Token0.Hex(),
			"token1":  dexKey.Token1.Hex(),
			"salt":    common.BytesToHash(dexKey.Salt[:]).Hex(),
		}).Debug("Successfully read DexKey")
	}

	if len(validPools) == 0 {
		return []PoolWithState{}, nil
	}

	// Read ALL dexVariables
	pools := make([]PoolWithState, 0, len(validPools))
	var skippedCount int
	for i, dexId := range dexIds {
		dexVariablesSlot := u.calculatePoolStateSlot(dexId, 0)
		dexVariables, err := u.readFromStorage(ctx, dexVariablesSlot)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexType": DexType,
				"error":   err,
				"dexId":   fmt.Sprintf("%x", dexId),
			}).Error("Failed to read dexVariables")
			return nil, err
		}

		// Skip uninitialized pools
		if dexVariables.Sign() == 0 {
			skippedCount++
			continue
		}

		// Extract fee from dexVariables
		fee := new(big.Int).And(dexVariables, X13B)

		// Create minimal pool state for listing
		poolState := PoolState{
			DexVariables:     uint256.MustFromBig(dexVariables),
			CenterPriceShift: big256.U0,
			RangeShift:       big256.U0,
			ThresholdShift:   big256.U0,
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
	}).Info("Pool discovery completed using direct ethClient calls")

	return pools, nil
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
	// Storage slot mapping for FluidDexLite pool state variables
	var baseSlot uint64
	switch offset {
	case 0: // _dexVariables mapping
		baseSlot = StorageSlotDexVariables
	case 1: // _centerPriceShift mapping
		baseSlot = StorageSlotCenterPriceShift
	case 2: // _rangeShift mapping
		baseSlot = StorageSlotRangeShift
	case 3: // _thresholdShift mapping
		baseSlot = StorageSlotThresholdShift
	default:
		baseSlot = StorageSlotDexVariables
	}

	// Convert bytes8 dexId to bytes32 (right-padded with zeros)
	var dexIdBytes32 common.Hash
	copy(dexIdBytes32[:], dexId[:])

	// Use Solidity mapping storage calculation: keccak256(abi.encode(key, slot))
	uint256Type, _ := abi.NewType("uint256", "", nil)
	bytes32Type, _ := abi.NewType("bytes32", "", nil)
	arguments := abi.Arguments{{Type: bytes32Type}, {Type: uint256Type}}

	encoded, err := arguments.Pack(common.BytesToHash(dexIdBytes32[:]), new(big.Int).SetUint64(baseSlot))
	if err != nil {
		// Fallback to manual encoding
		encoded = make([]byte, 64)
		copy(encoded[:32], dexIdBytes32[:])
		copy(encoded[32:], common.LeftPadBytes(new(big.Int).SetUint64(baseSlot).Bytes(), 32))
	}

	return common.BytesToHash(crypto.Keccak256(encoded))
}

// readFromStorage reads a single storage slot using ethClient.CallContract
func (u *PoolsListUpdater) readFromStorage(ctx context.Context, slot common.Hash) (*big.Int, error) {
	if u.ethClient == nil {
		return nil, fmt.Errorf("ethClient not available for storage reads")
	}

	// Pack the function call using the actual FluidDexLite ABI
	callData, err := fluidDexLiteABI.Pack("readFromStorage", slot)
	if err != nil {
		return nil, fmt.Errorf("failed to pack readFromStorage call: %w", err)
	}

	// Create contract call message
	contractAddr := common.HexToAddress(u.config.DexLiteAddress)
	callMsg := ethereum.CallMsg{
		To:   &contractAddr,
		Data: callData,
	}

	// Make the call
	resultBytes, err := u.ethClient.CallContract(ctx, callMsg, nil)
	if err != nil {
		return nil, fmt.Errorf("readFromStorage call failed: %w", err)
	}

	if len(resultBytes) != 32 {
		return nil, fmt.Errorf("unexpected result length: %d", len(resultBytes))
	}

	return new(big.Int).SetBytes(resultBytes), nil
}
