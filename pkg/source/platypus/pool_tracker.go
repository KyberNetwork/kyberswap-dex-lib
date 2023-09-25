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

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	ethClient *ethrpc.Client
}

func NewPoolTracker(ethClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		ethClient: ethClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
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

	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.Tokens = newPoolTokens(poolState.TokenAddresses)
	p.Timestamp = time.Now().Unix()

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
