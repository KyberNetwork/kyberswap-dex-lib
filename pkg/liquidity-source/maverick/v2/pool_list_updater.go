package maverickv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
)

type PoolsListUpdater struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

type PoolsListUpdaterMetadata struct {
	LastIndex *big.Int `json:"lastIndex"`
}

var _ = poollist.RegisterFactoryCE(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) *PoolsListUpdater {
	return &PoolsListUpdater{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}
}

// GetNewPools fetch new pools from the subgraph
func (u *PoolsListUpdater) GetNewPools(ctx context.Context, metadataBytes []byte) ([]entity.Pool, []byte, error) {
	var (
		dexID     = u.config.DexID
		startTime = time.Now()
	)
	logger.WithFields(logger.Fields{"dex_id": dexID}).Info("Started getting new pools")

	lastIndexPrevRound, err := u.getOffset(metadataBytes)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Warn("getOffset failed")
	}

	startIndex := new(big.Int).Add(lastIndexPrevRound, integer.One())
	endIndex := new(big.Int).Add(lastIndexPrevRound, big.NewInt(int64(u.config.NewPoolLimit)))

	factoryPools, err := u.getPoolsFromFactory(ctx, startIndex, endIndex)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("getPoolsFromFactory failed")
		return nil, metadataBytes, err
	}

	numberOfFactoryPools := len(factoryPools)

	if numberOfFactoryPools == 0 {
		logger.WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"startIndex":  startIndex,
				"pools_len":   0,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).Info("No new pools found")

		return nil, metadataBytes, nil
	}

	endIndex = new(big.Int).Add(startIndex, big.NewInt(int64(numberOfFactoryPools)))
	endIndex.Sub(endIndex, integer.One())

	logger.WithFields(
		logger.Fields{
			"dex_id":      dexID,
			"startIndex":  startIndex,
			"endIndex":    endIndex,
			"pools_len":   numberOfFactoryPools,
			"duration_ms": time.Since(startTime).Milliseconds(),
		},
	).Info("Finished getting pools from factory")

	pools, err := u.initPools(ctx, factoryPools)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("initPools failed")

		return nil, metadataBytes, err
	}

	newMetadataBytes, err := u.newMetadata(endIndex)
	if err != nil {
		logger.
			WithFields(logger.Fields{"dex_id": dexID, "err": err}).
			Error("newMetadata failed")

		return nil, metadataBytes, err
	}

	logger.
		WithFields(
			logger.Fields{
				"dex_id":      dexID,
				"pools_len":   len(pools),
				"lastIndex":   endIndex,
				"duration_ms": time.Since(startTime).Milliseconds(),
			},
		).
		Info("Finished getting new pools")

	return pools, newMetadataBytes, nil
}

// getOffset gets index of the last pool that is fetched
func (u *PoolsListUpdater) getOffset(metadataBytes []byte) (*big.Int, error) {
	if len(metadataBytes) == 0 {
		return integer.Zero(), nil
	}

	var metadata PoolsListUpdaterMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return integer.Zero(), err
	}

	return metadata.LastIndex, nil
}

func (u *PoolsListUpdater) getPoolsFromFactory(ctx context.Context, startIndex *big.Int, endIndex *big.Int) ([]common.Address, error) {
	var poolAddrs []common.Address

	lookupRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	lookupRequest.AddCall(&ethrpc.Call{
		ABI:    maverickV2FactoryABI,
		Target: u.config.FactoryAddress,
		Method: factoryMethodLookup,
		Params: []any{startIndex, endIndex},
	}, []any{&poolAddrs})

	if _, err := lookupRequest.Call(); err != nil {
		return nil, err
	}

	return poolAddrs, nil
}

// listPoolData receives list of pool addresses and returns their tokenA, tokenB and tick spacing
func (u *PoolsListUpdater) listPoolData(ctx context.Context, poolAddresses []common.Address) ([]common.Address, []common.Address, []*big.Int, error) {
	var (
		listTokenAResult = make([]common.Address, len(poolAddresses))
		listTokenBResult = make([]common.Address, len(poolAddresses))
		tickSpacingList  = make([]*big.Int, len(poolAddresses))
	)

	listDataRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i, pairAddress := range poolAddresses {
		listDataRequest.AddCall(&ethrpc.Call{
			ABI:    maverickV2PoolABI,
			Target: pairAddress.Hex(),
			Method: poolMethodTokenA,
			Params: nil,
		}, []any{&listTokenAResult[i]})

		listDataRequest.AddCall(&ethrpc.Call{
			ABI:    maverickV2PoolABI,
			Target: pairAddress.Hex(),
			Method: poolMethodTokenB,
			Params: nil,
		}, []any{&listTokenBResult[i]})

		listDataRequest.AddCall(&ethrpc.Call{
			ABI:    maverickV2PoolABI,
			Target: pairAddress.Hex(),
			Method: "tickSpacing",
			Params: nil,
		}, []any{&tickSpacingList[i]})
	}

	if _, err := listDataRequest.Aggregate(); err != nil {
		return nil, nil, nil, err
	}

	return listTokenAResult, listTokenBResult, tickSpacingList, nil
}

func (u *PoolsListUpdater) newMetadata(lastIndex *big.Int) ([]byte, error) {
	metadata := PoolsListUpdaterMetadata{
		LastIndex: lastIndex,
	}

	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return nil, err
	}

	return metadataBytes, nil
}

func (u *PoolsListUpdater) initPools(ctx context.Context, poolAddrs []common.Address) ([]entity.Pool, error) {
	tokenAList, tokenBList, tickSpacingList, err := u.listPoolData(ctx, poolAddrs)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolAddrs))

	for i, poolAddress := range poolAddrs {
		poolAddrLower := hexutil.Encode(poolAddress[:])

		token0 := &entity.PoolToken{
			Address:   hexutil.Encode(tokenAList[i][:]),
			Swappable: true,
		}

		token1 := &entity.PoolToken{
			Address:   hexutil.Encode(tokenBList[i][:]),
			Swappable: true,
		}

		// Create StaticExtra with data from both on-chain and API
		staticExtra := StaticExtra{
			TickSpacing: int32(tickSpacingList[i].Int64()),
		}

		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"pool_address": poolAddress.Hex(),
				"error":        err,
			}).Error("Failed to marshal static extra data")
			continue
		}

		var newPool = entity.Pool{
			Address:     poolAddrLower,
			Exchange:    u.config.DexID,
			Type:        DexType,
			Timestamp:   time.Now().Unix(),
			Reserves:    []string{"0", "0"},
			Tokens:      []*entity.PoolToken{token0, token1},
			StaticExtra: string(staticExtraBytes),
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
