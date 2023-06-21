package platypus

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/machinebox/graphql"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

type PoolsListUpdater struct {
	config        *Config
	graphqlClient *graphql.Client
	ethClient     *ethrpc.Client
}

func NewPoolsListUpdater(cfg *Config, ethClient *ethrpc.Client) *PoolsListUpdater {
	graphqlClient := graphqlpkg.NewWithTimeout(cfg.SubgraphAPI, graphQLRequestTimeout)

	return &PoolsListUpdater{
		config:        cfg,
		graphqlClient: graphqlClient,
		ethClient:     ethClient,
	}
}

func (p *PoolsListUpdater) GetNewPools(
	ctx context.Context, metadata []byte,
) ([]entity.Pool, []byte, error) {
	logger.Info("Getting new pools...")

	meta := Metadata{LastUpdate: "0"}
	if len(metadata) > 0 {
		err := json.Unmarshal(metadata, &meta)
		if err != nil {
			logger.WithFields(logger.Fields{
				"metadata": metadata,
				"error":    err,
			}).Errorf("Fail to marshal metadata")
			return nil, nil, err
		}
	}

	subgraphPools, err := p.getPoolAddresses(ctx, meta.LastUpdate)
	if err != nil {
		logger.WithFields(logger.Fields{
			"lastUpdate": meta.LastUpdate,
			"error":      err,
		}).Errorf("Fail to get pools from subgraph")
		return nil, nil, err
	}

	if len(subgraphPools) == 0 {
		return nil, metadata, nil
	}

	meta.LastUpdate = subgraphPools[len(subgraphPools)-1].LastUpdate
	addresses := make([]string, 0, len(subgraphPools))
	for _, pool := range subgraphPools {
		addresses = append(addresses, pool.ID)
	}

	pools, err := p.getPools(ctx, addresses)
	if err != nil {
		logger.WithFields(logger.Fields{
			"addresses": addresses,
			"error":     err,
		}).Errorf("Fail to get pools' information")
		return nil, nil, err
	}

	metadata, err = json.Marshal(meta)
	if err != nil {
		logger.WithFields(logger.Fields{
			"metadata": meta,
			"error":    err,
		}).Errorf("Fail to marshal metadata")
		return nil, nil, err
	}

	return pools, metadata, nil
}

func (p *PoolsListUpdater) getPoolAddresses(
	ctx context.Context,
	lastUpdate string,
) ([]SubgraphPool, error) {
	req := graphql.NewRequest(fmt.Sprintf(`{
		pools (
			where: {
				lastUpdate_gte: "%s"
			}
			orderBy: lastUpdate
			orderDirection: asc
		) {
			id
			lastUpdate
		}
	}`, lastUpdate))

	var response struct {
		Pools []SubgraphPool `json:"pools"`
	}
	if err := p.graphqlClient.Run(ctx, req, &response); err != nil {
		logger.WithFields(logger.Fields{
			"lastUpdate": lastUpdate,
			"error":      err,
		}).Errorf("Fail to query pools from subgraph")
		return nil, err
	}

	validPools := lo.Filter(response.Pools, func(p SubgraphPool, _ int) bool { return !strings.EqualFold(p.ID, addressZero) })
	return validPools, nil
}

func (p *PoolsListUpdater) getPools(
	ctx context.Context, addresses []string,
) ([]entity.Pool, error) {
	// Get all active pools' state.
	poolStates, err := p.getPoolStates(ctx, addresses)
	if err != nil {
		logger.WithFields(logger.Fields{
			"addresses": addresses,
			"error":     err,
		}).Errorf("Fail to get pools' state")
		return nil, err
	}

	// Get all asset addresses.
	poolAssetAddressesMap, err := p.getAssetAddresses(ctx, poolStates)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("Fail to get assets' address")
		return nil, err
	}

	// Get all asset states.
	poolAssetStatesMap, err := p.getAssetStates(ctx, poolAssetAddressesMap)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("Fail to get assets' state")
		return nil, err
	}

	// Get SAvax rate for staked avax pool.
	sAvaxRate, err := p.getSAvaxRate(ctx, addressStakedAvax)
	if err != nil {
		logger.WithFields(logger.Fields{
			"contractAddres": addressStakedAvax,
			"error":          err,
		}).Errorf("Fail to get savax rate")
		return nil, err
	}

	pools := make([]entity.Pool, 0, len(poolStates))
	for _, state := range poolStates {
		assetStates := poolAssetStatesMap[state.Address]
		extra := newExtra(state, assetStates, sAvaxRate)
		extraBytes, err := json.Marshal(extra)
		if err != nil {
			return nil, err
		}

		reserves := make([]string, 0, len(assetStates))
		for _, assetState := range assetStates {
			reserves = append(reserves, assetState.Cash.String())
		}

		pools = append(pools, entity.Pool{
			Address:      state.Address,
			ReserveUsd:   0,
			AmplifiedTvl: 0,
			SwapFee:      0,
			Exchange:     p.config.DexID,
			Type:         getPoolTypeByPriceOracle(strings.ToLower(state.PriceOracle.Hex())),
			Timestamp:    time.Now().Unix(),
			Reserves:     reserves,
			Extra:        string(extraBytes),
			StaticExtra:  "",
			Tokens:       newPoolTokens(state.TokenAddresses),
		})
	}

	return pools, nil
}

func (p *PoolsListUpdater) getPoolStates(
	ctx context.Context, addresses []string,
) ([]PoolState, error) {
	states := make([]PoolState, len(addresses))
	request := p.ethClient.NewRequest()
	for i, address := range addresses {
		request.
			AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: address,
				Method: poolMethodGetC1,
				Params: nil,
			}, []interface{}{&states[i].C1}).
			AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: address,
				Method: poolMethodGetHaircutRate,
				Params: nil,
			}, []interface{}{&states[i].HaircutRate}).
			AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: address,
				Method: poolMethodGetPriceOracle,
				Params: nil,
			}, []interface{}{&states[i].PriceOracle}).
			AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: address,
				Method: poolMethodGetRetentionRatio,
				Params: nil,
			}, []interface{}{&states[i].RetentionRatio}).
			AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: address,
				Method: poolMethodGetSlippageParamK,
				Params: nil,
			}, []interface{}{&states[i].SlippageParamK}).
			AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: address,
				Method: poolMethodGetSlippageParamN,
				Params: nil,
			}, []interface{}{&states[i].SlippageParamN}).
			AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: address,
				Method: poolMethodGetTokenAddresses,
				Params: nil,
			}, []interface{}{&states[i].TokenAddresses}).
			AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: address,
				Method: poolMethodGetXThreshold,
				Params: nil,
			}, []interface{}{&states[i].XThreshold}).
			AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: address,
				Method: poolMethodPaused,
				Params: nil,
			}, []interface{}{&states[i].Paused})
	}

	if _, err := request.Aggregate(); err != nil {
		return nil, err
	}

	// Ignore paused pools.
	poolStates := make([]PoolState, 0, len(states))
	for i, state := range states {
		if state.Paused {
			continue
		}

		state.Address = addresses[i]
		poolStates = append(poolStates, state)
	}

	return poolStates, nil
}

func (p *PoolsListUpdater) getAssetAddresses(
	ctx context.Context, poolStates []PoolState,
) (map[string][]common.Address, error) {
	request := p.ethClient.NewRequest()
	poolAssetAddressesMap := make(map[string][]common.Address)
	for _, state := range poolStates {
		assetAddresses := make([]common.Address, len(state.TokenAddresses))
		poolAssetAddressesMap[state.Address] = assetAddresses
		for i, tokenAddress := range state.TokenAddresses {
			request.AddCall(&ethrpc.Call{
				ABI:    poolABI,
				Target: state.Address,
				Method: poolMethodAssetOf,
				Params: []interface{}{tokenAddress},
			}, []interface{}{&assetAddresses[i]})
		}
	}

	if _, err := request.Aggregate(); err != nil {
		return nil, err
	}

	return poolAssetAddressesMap, nil
}

func (p *PoolsListUpdater) getAssetStates(
	ctx context.Context, poolAssetAddressesMap map[string][]common.Address,
) (map[string][]AssetState, error) {
	request := p.ethClient.NewRequest()
	poolAssetStatesMap := make(map[string][]AssetState)
	for poolAddress, assetAddresses := range poolAssetAddressesMap {
		assetStates := make([]AssetState, len(assetAddresses))
		poolAssetStatesMap[poolAddress] = assetStates
		for i, assetAddress := range assetAddresses {
			address := assetAddress.Hex()
			request.
				AddCall(&ethrpc.Call{
					ABI:    assetABI,
					Target: address,
					Method: assetMethodCash,
					Params: nil,
				}, []interface{}{&assetStates[i].Cash}).
				AddCall(&ethrpc.Call{
					ABI:    assetABI,
					Target: address,
					Method: assetMethodDecimals,
					Params: nil,
				}, []interface{}{&assetStates[i].Decimals}).
				AddCall(&ethrpc.Call{
					ABI:    assetABI,
					Target: address,
					Method: assetMethodLiability,
					Params: nil,
				}, []interface{}{&assetStates[i].Liability}).
				AddCall(&ethrpc.Call{
					ABI:    assetABI,
					Target: address,
					Method: assetMethodUnderlyingToken,
					Params: nil,
				}, []interface{}{&assetStates[i].UnderlyingToken}).
				AddCall(&ethrpc.Call{
					ABI:    assetABI,
					Target: address,
					Method: assetMethodAggregateAccount,
				}, []interface{}{&assetStates[i].AggregateAccount})
		}
	}

	if _, err := request.Aggregate(); err != nil {
		return nil, err
	}

	return poolAssetStatesMap, nil
}

func (p *PoolsListUpdater) getSAvaxRate(ctx context.Context, address string) (*big.Int, error) {
	var rate *big.Int
	request := p.ethClient.NewRequest().
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
