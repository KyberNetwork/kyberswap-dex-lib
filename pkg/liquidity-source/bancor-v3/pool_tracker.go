package bancorv3

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrPoolCollectionNotFound   = errors.New("pool collection not found")
	ErrCollectionByPoolNotFound = errors.New("collection by pool not found")
	ErrPoolDataNotFound         = errors.New("pool data not found")
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params poolpkg.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, poolpkg.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"dexId":       t.config.DexID,
		"dexType":     DexType,
		"poolAddress": p.Address,
	}).Info("Start updating state ...")

	defer func() {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Info("Finish updating state.")
	}()

	liquidityPools, blockNbr, err := t.getLiquidityPools(ctx, p.Address, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	collectionByPool, err := t.getCollectionByPool(ctx, blockNbr, p.Address, liquidityPools, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	poolCollections, err := t.getPoolCollections(ctx, blockNbr, collectionByPool, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	if err := t.updatePool(ctx, &p, blockNbr, collectionByPool, poolCollections); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	return p, nil
}

func (t *PoolTracker) updatePool(
	_ context.Context,
	p *entity.Pool,
	blockNbr *big.Int,
	collectionByPool map[string]string,
	poolCollections map[string]*poolCollectionResp,
) error {
	var (
		nativeIdx = -1
		tokens    = []*entity.PoolToken{}
		reserves  = entity.PoolReserves{}
		poolCols  = map[string]*poolCollection{}
		colByPool = map[string]string{}
	)

	// pool collections
	for pcAddr, pc := range poolCollections {
		poolData := make(map[string]*pool)

		for poolAddr, poolDat := range pc.PoolData {
			if !poolDat.TradingEnabled {
				continue
			}

			var (
				poolToken                    = strings.ToLower(poolDat.PoolToken.Hex())
				bntTradingLiquidity, _       = uint256.FromBig(poolDat.PoolLiquidity.BntTradingLiquidity)
				baseTokenTradingLiquidity, _ = uint256.FromBig(poolDat.PoolLiquidity.BaseTokenTradingLiquidity)
				stakedBalance, _             = uint256.FromBig(poolDat.PoolLiquidity.StakedBalance)
			)

			pool := pool{
				PoolToken:      poolToken,
				TradingFeePPM:  uint256.NewInt(uint64(poolDat.TradingFeePPM)),
				TradingEnabled: poolDat.TradingEnabled,
				Liquidity: &poolLiquidity{
					BNTTradingLiquidity:       bntTradingLiquidity,
					BaseTokenTradingLiquidity: baseTokenTradingLiquidity,
					StakedBalance:             stakedBalance,
				},
			}
			poolData[poolAddr] = &pool
		}

		poolCols[pcAddr] = &poolCollection{
			NetworkFeePMM: uint256.NewInt(uint64(pc.NetworkFeePMM)),
			BNT:           strings.ToLower(t.config.BNT),
			PoolData:      poolData,
		}
	}

	// collection by pool
	for pool, col := range collectionByPool {
		poolData, ok := poolCols[col].PoolData[pool]
		if !ok {
			continue
		}
		colByPool[pool] = col
		reserves = append(reserves, poolData.Liquidity.StakedBalance.String())
		if strings.EqualFold(pool, valueobject.EtherAddress) {
			nativeIdx = len(tokens)
			tokens = append(tokens, &entity.PoolToken{
				Address:   strings.ToLower(valueobject.WETHByChainID[t.config.ChainID]),
				Swappable: true,
			})

		} else {
			tokens = append(tokens, &entity.PoolToken{
				Address:   pool,
				Swappable: true,
			})
		}
	}

	// update data
	p.Tokens = tokens
	p.Reserves = reserves
	extraBytes, err := json.Marshal(Extra{
		NativeIdx:        nativeIdx,
		CollectionByPool: colByPool,
		PoolCollections:  poolCols,
	})
	if err != nil {
		return err
	}
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNbr.Uint64()

	return nil
}

func (t *PoolTracker) getPoolCollections(
	ctx context.Context,
	blockNbr *big.Int,
	collectionByPool map[string]string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (map[string]*poolCollectionResp, error) {
	ret := map[string]*poolCollectionResp{}
	poolsByPoolCollection := t.groupPoolsByPoolCollection(collectionByPool)
	for poolCollectionAddr, pools := range poolsByPoolCollection {
		poolCollection, err := t.getPoolCollection(
			ctx,
			blockNbr,
			poolCollectionAddr,
			pools,
			overrides,
		)
		if err != nil {
			return nil, err
		}
		ret[poolCollectionAddr] = poolCollection
	}
	return ret, nil
}

func (t *PoolTracker) getPoolCollection(
	ctx context.Context,
	blockNbr *big.Int,
	poolCollection string,
	pools []string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*poolCollectionResp, error) {
	req := t.ethrpcClient.R().
		SetContext(ctx).
		SetBlockNumber(blockNbr)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	poolDatResp := make([]*poolDataResp, len(pools))
	for idx, p := range pools {
		poolDatResp[idx] = &poolDataResp{}

		req.AddCall(&ethrpc.Call{
			ABI:    poolCollectionABI,
			Target: poolCollection,
			Method: poolCollectionMethodPoolData,
			Params: []interface{}{common.HexToAddress(p)},
		}, []interface{}{&poolDatResp[idx]})
	}

	var fee uint32
	req.AddCall(&ethrpc.Call{
		ABI:    poolCollectionABI,
		Target: poolCollection,
		Method: poolCollectionMethodNetworkFeePPM,
	}, []interface{}{&fee})

	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}

	poolData := map[string]*poolDataResp{}
	for idx, pool := range pools {
		poolData[pool] = poolDatResp[idx]
	}

	return &poolCollectionResp{
		PoolData:      poolData,
		NetworkFeePMM: fee,
	}, nil
}

func (t *PoolTracker) getCollectionByPool(
	ctx context.Context,
	blockNbr *big.Int,
	bancorNetworkAddress string,
	liquidityPools []string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (map[string]string, error) {
	req := t.ethrpcClient.R().
		SetContext(ctx).
		SetBlockNumber(blockNbr)
	if overrides != nil {
		req.SetOverrides(overrides)
	}
	poolCollections := make([]common.Address, len(liquidityPools))
	for idx, liquidityPool := range liquidityPools {
		req.AddCall(&ethrpc.Call{
			ABI:    bancorNetworkABI,
			Target: bancorNetworkAddress,
			Method: bancorNetworkMethodCollectionByPool,
			Params: []interface{}{common.HexToAddress(liquidityPool)},
		}, []interface{}{&poolCollections[idx]})
	}
	_, err := req.Aggregate()
	if err != nil {
		return nil, err
	}

	poolByCollection := make(map[string]string)
	for idx, liquidityPool := range liquidityPools {
		poolByCollection[liquidityPool] = strings.ToLower(poolCollections[idx].Hex())
	}

	return poolByCollection, nil
}

func (t *PoolTracker) getLiquidityPools(
	ctx context.Context,
	bancorNetworkAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]string, *big.Int, error) {
	var addresses []common.Address
	req := t.ethrpcClient.R().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    bancorNetworkABI,
		Target: bancorNetworkAddress,
		Method: bancorNetworkMethodLiquidityPools,
	}, []interface{}{&addresses})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, err
	}

	ret := make([]string, 0, len(addresses))
	for _, addr := range addresses {
		ret = append(ret, strings.ToLower(addr.Hex()))
	}

	return ret, res.BlockNumber, nil
}

func (t *PoolTracker) groupPoolsByPoolCollection(collectionByPool map[string]string) map[string][]string {
	ret := map[string][]string{}
	for token, poolCollection := range collectionByPool {
		ret[poolCollection] = append(ret[poolCollection], token)
	}
	return ret
}
