package maverickv2

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util"
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

	ctx = util.NewContextWithTimestamp(ctx)

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
		Params: []interface{}{startIndex, endIndex},
	}, []interface{}{&poolAddrs})

	if _, err := lookupRequest.Call(); err != nil {
		return nil, err
	}

	return poolAddrs, nil
}

// listPoolTokens receives list of pool addresses and returns their tokenA and tokenB
func (u *PoolsListUpdater) listPoolTokens(ctx context.Context, poolAddresses []common.Address) ([]common.Address, []common.Address, error) {
	var (
		listTokenAResult = make([]common.Address, len(poolAddresses))
		listTokenBResult = make([]common.Address, len(poolAddresses))
	)

	listTokensRequest := u.ethrpcClient.NewRequest().SetContext(ctx)

	for i, pairAddress := range poolAddresses {
		listTokensRequest.AddCall(&ethrpc.Call{
			ABI:    maverickV2PoolABI,
			Target: pairAddress.Hex(),
			Method: poolMethodTokenA,
			Params: nil,
		}, []interface{}{&listTokenAResult[i]})

		listTokensRequest.AddCall(&ethrpc.Call{
			ABI:    maverickV2PoolABI,
			Target: pairAddress.Hex(),
			Method: poolMethodTokenB,
			Params: nil,
		}, []interface{}{&listTokenBResult[i]})
	}

	if _, err := listTokensRequest.Aggregate(); err != nil {
		return nil, nil, err
	}

	return listTokenAResult, listTokenBResult, nil
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
	tokenAList, tokenBList, err := u.listPoolTokens(ctx, poolAddrs)
	if err != nil {
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolAddrs))

	for i, poolAddress := range poolAddrs {
		token0 := &entity.PoolToken{
			Address:   strings.ToLower(tokenAList[i].Hex()),
			Swappable: true,
		}

		token1 := &entity.PoolToken{
			Address:   strings.ToLower(tokenBList[i].Hex()),
			Swappable: true,
		}

		var newPool = entity.Pool{
			Address:   strings.ToLower(poolAddress.Hex()),
			Exchange:  u.config.DexID,
			Type:      DexType,
			Timestamp: time.Now().Unix(),
			Reserves:  []string{"0", "0"},
			Tokens:    []*entity.PoolToken{token0, token1},
		}

		pools = append(pools, newPool)
	}

	return pools, nil
}
