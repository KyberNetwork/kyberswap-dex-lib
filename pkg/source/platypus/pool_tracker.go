package platypus

import (
	"context"
	"encoding/json"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolTracker struct {
	ethClient *ethrpc.Client
}

func NewPoolTracker(ethClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		ethClient: ethClient,
	}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	logger.WithFields(logger.Fields{
		"address": p.Address,
	}).Infof("[Platypus] Start getting new pool's state")

	// Get pool's state.
	poolState, err := t.getPoolState(ctx, p.Address)
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"error":   err,
		}).Errorf("Fail to get pool's state")
		return entity.Pool{}, err
	}

	// Get assets' address.
	assetAddresses, err := t.getAssetAddresses(ctx, p.Address, poolState.TokenAddresses)
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"tokens":  poolState.TokenAddresses,
			"error":   err,
		}).Errorf("Fail to get address of assets")
		return entity.Pool{}, err
	}

	// Get assets' state.
	assetStates, err := t.getAssetStates(ctx, assetAddresses)
	if err != nil {
		logger.WithFields(logger.Fields{
			"assetAddresses": assetAddresses,
			"error":          err,
		}).Errorf("Fail to get asset states")
		return entity.Pool{}, err
	}

	p.Type = getPoolTypeByPriceOracle(strings.ToLower(poolState.PriceOracle.Hex()))

	sAvaxRate := big.NewInt(0)
	if p.Type == poolTypePlatypusAvax {
		sAvaxRate, err = t.getSAvaxRate(ctx, addressStakedAvax)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("Fail to get staked avax rate")
			return entity.Pool{}, err
		}
	}

	extra := newExtra(poolState, assetStates, sAvaxRate)

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"address": p.Address,
			"extra":   extra,
			"error":   err,
		}).Errorf("Fail to marshal pool's extra")
		return entity.Pool{}, err
	}

	reserves := make([]string, 0, len(assetStates))
	for _, assetState := range assetStates {
		reserves = append(reserves, assetState.Cash.String())
	}

	// dependency tracking:
	// platypus-pure: all assetAddress
	// platypus-base (chain-link): all assetAddress, priceOracleAddress, aggregatorAddress
	// platypus-avax (and similar): all assetAddress, priceOracleAddress
	dependencies := lo.Map(assetAddresses, func(a common.Address, _ int) string { return a.Hex() })
	switch p.Type {
	case poolTypePlatypusBase:
		dependencies = append(dependencies, poolState.PriceOracle.Hex())
		// get aggregators for chainlink pools (platypus-base)
		aggregators, err := t.getChainlinkProxyAggregator(ctx, poolState)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("Fail to get chainlink pool aggregator")
			return entity.Pool{}, err
		}
		for _, ag := range aggregators {
			agAdr := ag.Hex()
			if !strings.EqualFold(agAdr, addressZero) {
				dependencies = append(dependencies, agAdr)
			}
		}
	case poolTypePlatypusAvax:
		dependencies = append(dependencies, poolState.PriceOracle.Hex())
	}

	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.Tokens = newPoolTokens(poolState.TokenAddresses)
	p.Timestamp = time.Now().Unix()
	p.Dependencies = dependencies

	return p, nil
}

func (t *PoolTracker) getPoolState(ctx context.Context, address string) (PoolState, error) {
	var state PoolState
	request := t.ethClient.NewRequest().
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetC1,
			Params: nil,
		}, []interface{}{&state.C1}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetHaircutRate,
			Params: nil,
		}, []interface{}{&state.HaircutRate}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetPriceOracle,
			Params: nil,
		}, []interface{}{&state.PriceOracle}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetRetentionRatio,
			Params: nil,
		}, []interface{}{&state.RetentionRatio}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetSlippageParamK,
			Params: nil,
		}, []interface{}{&state.SlippageParamK}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetSlippageParamN,
			Params: nil,
		}, []interface{}{&state.SlippageParamN}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetTokenAddresses,
			Params: nil,
		}, []interface{}{&state.TokenAddresses}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodGetXThreshold,
			Params: nil,
		}, []interface{}{&state.XThreshold}).
		AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: address,
			Method: poolMethodPaused,
			Params: nil,
		}, []interface{}{&state.Paused})

	if _, err := request.Aggregate(); err != nil {
		return PoolState{}, err
	}

	return state, nil
}

func (t *PoolTracker) getAssetAddresses(
	ctx context.Context,
	poolAddress string,
	tokenAddresses []common.Address,
) ([]common.Address, error) {
	assetAddresses := make([]common.Address, len(tokenAddresses))
	request := t.ethClient.NewRequest()
	for i, tokenAddress := range tokenAddresses {
		request.AddCall(&ethrpc.Call{
			ABI:    poolABI,
			Target: poolAddress,
			Method: poolMethodAssetOf,
			Params: []interface{}{tokenAddress},
		}, []interface{}{&assetAddresses[i]})
	}

	if _, err := request.Aggregate(); err != nil {
		return nil, err
	}

	return assetAddresses, nil
}

func (t *PoolTracker) getAssetStates(
	ctx context.Context, addresses []common.Address,
) ([]AssetState, error) {
	states := make([]AssetState, len(addresses))
	request := t.ethClient.NewRequest()
	for i, addr := range addresses {
		address := addr.Hex()
		request.
			AddCall(&ethrpc.Call{
				ABI:    assetABI,
				Target: address,
				Method: assetMethodCash,
				Params: nil,
			}, []interface{}{&states[i].Cash}).
			AddCall(&ethrpc.Call{
				ABI:    assetABI,
				Target: address,
				Method: assetMethodDecimals,
				Params: nil,
			}, []interface{}{&states[i].Decimals}).
			AddCall(&ethrpc.Call{
				ABI:    assetABI,
				Target: address,
				Method: assetMethodLiability,
				Params: nil,
			}, []interface{}{&states[i].Liability}).
			AddCall(&ethrpc.Call{
				ABI:    assetABI,
				Target: address,
				Method: assetMethodUnderlyingToken,
				Params: nil,
			}, []interface{}{&states[i].UnderlyingToken}).
			AddCall(&ethrpc.Call{
				ABI:    assetABI,
				Target: address,
				Method: assetMethodAggregateAccount,
			}, []interface{}{&states[i].AggregateAccount})
	}

	if _, err := request.Aggregate(); err != nil {
		return nil, err
	}

	return states, nil
}

func (t *PoolTracker) getSAvaxRate(ctx context.Context, address string) (*big.Int, error) {
	var rate *big.Int
	request := t.ethClient.NewRequest().
		AddCall(&ethrpc.Call{
			ABI:    stakedAvaxABI,
			Target: address,
			Method: stakedAvaxMethodGetPooledAvaxByShares,
			Params: []interface{}{bOne},
		}, []interface{}{&rate})
	if _, err := request.Call(); err != nil {
		return nil, err
	}

	return rate, nil
}

func (p *PoolTracker) getChainlinkProxyAggregator(
	ctx context.Context, state PoolState,
) ([]common.Address, error) {
	logger.WithFields(logger.Fields{"pool": state}).Debug("get chainlink proxy")

	// first get the proxies
	request := p.ethClient.NewRequest()
	proxyAddresses := make([]common.Address, len(state.TokenAddresses))
	for i, tokenAddress := range state.TokenAddresses {
		request.AddCall(&ethrpc.Call{
			ABI:    oracleABI,
			Target: state.PriceOracle.Hex(),
			Method: poolMethodSourceAsset,
			Params: []interface{}{tokenAddress},
		}, []interface{}{&proxyAddresses[i]})
	}

	if _, err := request.TryAggregate(); err != nil {
		return nil, err
	}

	logger.WithFields(logger.Fields{"pool": state, "proxies": proxyAddresses}).Debug("get chainlink proxy")

	// then get the aggregators of those proxies
	request = p.ethClient.NewRequest()

	invalidProxy := false
	for i := range state.TokenAddresses {
		if proxyAddresses[i].Hex() == addressZero {
			invalidProxy = true
			break
		}
	}
	if invalidProxy {
		logger.WithFields(logger.Fields{"pool": state.Address}).Info("ignore invalid proxy")
		return nil, nil
	}

	aggregatorAddresses := make([]common.Address, len(state.TokenAddresses))
	for i := range state.TokenAddresses {
		request.AddCall(&ethrpc.Call{
			ABI:    chainlinkABI,
			Target: proxyAddresses[i].Hex(),
			Method: poolMethodAggregator,
			Params: []interface{}{},
		}, []interface{}{&aggregatorAddresses[i]})
	}

	if _, err := request.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{"pool": state.Address, "error": err}).Info("ignore invalid proxy aggregator")
		return nil, nil
	}

	logger.WithFields(logger.Fields{"pool": state, "aggregators": aggregatorAddresses}).Debug("get chainlink aggregator")

	return aggregatorAddresses, nil
}
