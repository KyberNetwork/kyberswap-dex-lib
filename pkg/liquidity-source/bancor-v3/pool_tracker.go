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

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var (
	ErrPoolCollectionNotFound = errors.New("pool collection not found")
	ErrPoolDataNotFound       = errors.New("pool data not found")
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
	_ poolpkg.GetNewPoolStateParams,
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

	liquidityPools, blockNbr, err := t.getLiquidityPools(ctx, p.Address)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	// get collection by pool
	collectionByPool, err := t.getCollectionByPool(ctx, blockNbr, p.Address, liquidityPools)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	poolCollections, err := t.getPoolCollections(ctx, blockNbr, collectionByPool)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	if err := t.updateTokensAndReserves(ctx, &p, liquidityPools, collectionByPool, poolCollections); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	if err := t.updatePoolCollections(ctx, &p, collectionByPool, poolCollections); err != nil {
		logger.WithFields(logger.Fields{
			"dexId":       t.config.DexID,
			"dexType":     DexType,
			"poolAddress": p.Address,
		}).Error(err.Error())
		return p, err
	}

	return p, nil
}

func (t *PoolTracker) updatePoolCollections(
	ctx context.Context,
	p *entity.Pool,
	collectionByPool map[string]string,
	poolCollections map[string]*poolCollectionResp,
) error {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return err
	}
	extra.CollectionByPool = collectionByPool

	poolCols := make(map[string]*poolCollection)
	for pcAddr, pc := range poolCollections {
		poolData := make(map[string]*pool)
		for poolAddr, poolDat := range pc.PoolData {
			poolData[poolAddr] = &pool{
				PoolToken:      poolDat.PoolToken.Hex(),
				TradingFeePPM:  uint256.NewInt(uint64(poolDat.TradingFeePPM)),
				TradingEnabled: poolDat.TradingEnabled,
				Liquidity: poolLiquidity{
					BNTTradingLiquidity:       uint256.MustFromBig(poolDat.PoolLiquidity.BntTradingLiquidity),
					BaseTokenTradingLiquidity: uint256.MustFromBig(poolDat.PoolLiquidity.BaseTokenTradingLiquidity),
					StakedBalance:             uint256.MustFromBig(poolDat.PoolLiquidity.StakedBalance),
				},
			}
		}

		poolCols[pcAddr] = &poolCollection{
			NetworkFeePMM: uint256.NewInt(uint64(pc.NetworkFeePMM)),
			BNT:           t.config.BNT,
			PoolData:      poolData,
		}
	}
	extra.PoolCollections = poolCols

	newExtraBytes, err := json.Marshal(extra)
	if err != nil {
		return err
	}
	p.Extra = string(newExtraBytes)

	return nil
}

func (t *PoolTracker) updateTokensAndReserves(
	ctx context.Context,
	p *entity.Pool,
	liquidityPools []string,
	collectionByPool map[string]string,
	poolCollections map[string]*poolCollectionResp,
) error {
	exists := map[string]struct{}{}
	for _, token := range p.Tokens {
		exists[token.Address] = struct{}{}
	}

	for _, liquidityPool := range liquidityPools {
		if _, ok := exists[liquidityPool]; ok {
			continue
		}
		p.Tokens = append(p.Tokens, &entity.PoolToken{Address: liquidityPool})
		p.Reserves = append(p.Reserves, "0")
	}

	for idx, token := range p.Tokens {
		poolCollectionAddr, ok := collectionByPool[token.Address]
		if !ok {
			return ErrPoolCollectionNotFound
		}
		poolCollection, ok := poolCollections[poolCollectionAddr]
		if !ok {
			return ErrPoolCollectionNotFound
		}
		poolData, ok := poolCollection.PoolData[token.Address]
		if !ok {
			return ErrPoolDataNotFound
		}
		p.Reserves[idx] = poolData.PoolLiquidity.StakedBalance.String()
	}

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return err
	}
	if extra.NativeIdx < 0 {
		for idx, token := range p.Tokens {
			if strings.EqualFold(token.Address, valueobject.EtherAddress) {
				extra.NativeIdx = idx
				p.Tokens[idx].Address = valueobject.WETHByChainID[t.config.ChainID]
				break
			}
		}
		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return err
		}
		p.Extra = string(extraBytes)
	}

	return nil
}

func (t *PoolTracker) getPoolCollections(
	ctx context.Context,
	blockNbr *big.Int,
	collectionByPool map[string]string,
) (map[string]*poolCollectionResp, error) {
	ret := map[string]*poolCollectionResp{}
	poolsByPoolCollection := t.groupPoolsByPoolCollection(collectionByPool)
	for poolCollectionAddr, pools := range poolsByPoolCollection {
		poolCollection, err := t.getPoolCollection(
			ctx,
			blockNbr,
			poolCollectionAddr,
			pools,
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
) (*poolCollectionResp, error) {
	req := t.ethrpcClient.R()

	poolDatResp := make([]*poolDataResp, len(pools))
	for idx, p := range pools {
		req.AddCall(&ethrpc.Call{
			ABI:    poolCollectionABI,
			Target: poolCollection,
			Method: poolCollectionMethodPoolData,
			Params: []interface{}{common.HexToAddress(p)},
		}, []interface{}{poolDatResp[idx]})
	}

	var fee uint32
	req.AddCall(&ethrpc.Call{
		ABI:    poolCollectionABI,
		Target: poolCollection,
		Method: poolCollectionMethodPoolData,
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
) (map[string]string, error) {
	req := t.ethrpcClient.R()
	poolCollections := make([]string, len(liquidityPools))
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
		poolByCollection[liquidityPool] = poolCollections[idx]
	}

	return poolByCollection, nil
}

func (t *PoolTracker) getLiquidityPools(
	ctx context.Context,
	bancorNetworkAddress string,
) ([]string, *big.Int, error) {
	var addresses []common.Address
	req := t.ethrpcClient.R()
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
