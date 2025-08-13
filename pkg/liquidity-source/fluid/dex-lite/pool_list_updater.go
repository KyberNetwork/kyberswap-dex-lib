package dexLite

import (
	"context"
	"encoding/hex"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolsListUpdater struct {
	config       Config
	ethrpcClient *ethrpc.Client
}

type Metadata struct {
	LastSyncPoolsLength uint64 `json:"lastSyncPoolsLength"`
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

	var metadata Metadata
	_ = json.Unmarshal(metadataBytes, &metadata)
	// only handle new pools after last synced index
	nextDexKeys, err := u.getNextDexKeys(ctx, metadata.LastSyncPoolsLength)
	if err != nil {
		return nil, nil, err
	} else if len(nextDexKeys) == 0 {
		return nil, metadataBytes, nil
	}

	metadata.LastSyncPoolsLength += uint64(len(nextDexKeys))
	newMetadataBytes, _ := json.Marshal(metadata)

	pools := make([]entity.Pool, 0, len(nextDexKeys))
	for _, dexKey := range nextDexKeys {
		dexId := u.calculateDexId(dexKey)
		staticExtraBytes, err := json.Marshal(StaticExtra{
			DexLiteAddress: u.config.DexLiteAddress,
			DexKey:         *dexKey,
			DexId:          dexId,
		})
		if err != nil {
			logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("failed to marshal StaticExtra")
			return nil, nil, err
		}

		// Store only the essential FluidDexLite data
		extraBytes, err := json.Marshal(PoolExtraMarshal{
			BlockTimestamp: uint64(time.Now().Unix()),
		})
		if err != nil {
			logger.WithFields(logger.Fields{"dexType": DexType, "error": err}).Error("failed to marshal PoolExtra")
			return nil, nil, err
		}

		tokens := []*entity.PoolToken{
			{
				Address:   valueobject.WrapNativeLower(hexutil.Encode(dexKey.Token0[:]), u.config.ChainID),
				Swappable: true,
			},
			{
				Address:   valueobject.WrapNativeLower(hexutil.Encode(dexKey.Token1[:]), u.config.ChainID),
				Swappable: true,
			},
		}

		pools = append(pools, entity.Pool{
			Address:     strings.ToLower(u.config.DexLiteAddress) + hex.EncodeToString(dexId[:]),
			Exchange:    valueobject.ExchangeFluidDexLite,
			Type:        DexType,
			Reserves:    entity.PoolReserves{"0", "0"},
			Tokens:      tokens,
			Extra:       string(extraBytes),
			StaticExtra: string(staticExtraBytes),
		})
	}

	return pools, newMetadataBytes, nil
}

func (u *PoolsListUpdater) getNextDexKeys(ctx context.Context, from uint64) ([]*DexKey, error) {
	// Get the number of dex keys in the dex list
	var dexListLength *big.Int

	_, err := u.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    fluidDexLiteABI,
		Target: u.config.DexLiteAddress,
		Method: SRMethodReadFromStorage,
		Params: []any{StorageSlotDexesList},
	}, []any{&dexListLength}).Call()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("Failed to get dex list length")
		return nil, err
	}

	length := dexListLength.Uint64()
	if length <= from {
		return nil, nil
	}

	return u.readDexKeys(ctx, from, length)
}

// readDexKeys reads all dex keys using multicall
func (u *PoolsListUpdater) readDexKeys(ctx context.Context, from, till uint64) ([]*DexKey, error) {
	var dexKeys []*DexKey
	if till-from > MaxBatchSize {
		dexKeys = make([]*DexKey, 0, till-from)
		for i := from; i < till; i += MaxBatchSize {
			newPools, err := u.readDexKeys(ctx, i, min(till, i+MaxBatchSize))
			if err != nil {
				break
			}
			dexKeys = append(dexKeys, newPools...)
		}
		return dexKeys, nil
	}

	dexKeyFns := make([]func() *DexKey, till-from)
	req := u.ethrpcClient.NewRequest().SetContext(ctx)
	for i := from; i < till; i++ {
		// Read 3 consecutive slots for this DexKey struct
		token0Slot := u.calculateArraySlot(1, i)
		var tmp1, tmp2 big.Int
		token1Slot := tmp1.Add(token0Slot, bignumber.One)
		saltSlot := tmp2.Add(token0Slot, bignumber.Two)

		// Read token0, token1, salt
		req.AddCall(&ethrpc.Call{
			ABI:    fluidDexLiteABI,
			Target: u.config.DexLiteAddress,
			Method: SRMethodReadFromStorage,
			Params: []any{common.BigToHash(token0Slot)},
		}, []any{&token0Slot}).AddCall(&ethrpc.Call{
			ABI:    fluidDexLiteABI,
			Target: u.config.DexLiteAddress,
			Method: SRMethodReadFromStorage,
			Params: []any{common.BigToHash(token1Slot)},
		}, []any{&token1Slot}).AddCall(&ethrpc.Call{
			ABI:    fluidDexLiteABI,
			Target: u.config.DexLiteAddress,
			Method: SRMethodReadFromStorage,
			Params: []any{common.BigToHash(saltSlot)},
		}, []any{&saltSlot})

		dexKeyFns[i-from] = func() *DexKey {
			if token0Slot.Sign() == 0 || token1Slot.Sign() == 0 {
				return nil // Skip invalid dexKeys (token0 == 0 || token1 == 0)
			}
			// Reconstruct DexKey
			dexKey := &DexKey{
				Token0: common.BigToAddress(token0Slot),
				Token1: common.BigToAddress(token1Slot),
				Salt:   common.BigToHash(saltSlot),
			}
			logger.WithFields(logger.Fields{
				"dexType": DexType,
				"index":   i,
				"token0":  dexKey.Token0,
				"token1":  dexKey.Token1,
				"salt":    dexKey.Salt,
			}).Debug("successfully read DexKey")
			return dexKey
		}
	}
	if _, err := req.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"dexType": DexType,
			"error":   err,
		}).Error("failed to read DexKey slots")
		return nil, err
	}

	dexKeys = make([]*DexKey, 0, len(dexKeyFns))
	for _, dexKeyFn := range dexKeyFns {
		if dexKey := dexKeyFn(); dexKey != nil {
			dexKeys = append(dexKeys, dexKey)
		}
	}
	if len(dexKeys) == 0 {
		return nil, nil
	}

	logger.WithFields(logger.Fields{
		"dexType": DexType,
		"from":    from,
		"till":    till,
		"valid":   len(dexKeys),
	}).Info("got dexKeys")

	return dexKeys, nil
}

// Helper functions for storage calculations
func (u *PoolsListUpdater) calculateArraySlot(baseSlot, index uint64) *big.Int {
	// For dynamic arrays: keccak256(baseSlot) + index
	var tmp uint256.Int
	return tmp.SetBytes(crypto.Keccak256(tmp.SetUint64(baseSlot).PaddedBytes(32))).
		AddUint64(&tmp, index).ToBig()
}

func (u *PoolsListUpdater) calculateDexId(dexKey *DexKey) DexId {
	// dexId = bytes8(keccak256(abi.encode(dexKey)))
	hash := crypto.Keccak256(addressPadding, dexKey.Token0[:],
		addressPadding, dexKey.Token1[:], dexKey.Salt[:])
	return DexId(hash[:8])
}
