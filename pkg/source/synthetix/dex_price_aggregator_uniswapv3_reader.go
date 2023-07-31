package synthetix

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

type DexPriceAggregatorUniswapV3Reader struct {
	abi          abi.ABI
	cfg          *Config
	ethrpcClient *ethrpc.Client
}

func NewDexPriceAggregatorUniswapV3Reader(cfg *Config, ethrpcClient *ethrpc.Client) *DexPriceAggregatorUniswapV3Reader {
	return &DexPriceAggregatorUniswapV3Reader{
		abi:          dexPriceAggregatorUniswapV3,
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
	}
}

func (r *DexPriceAggregatorUniswapV3Reader) Read(
	ctx context.Context,
	poolState *PoolState,
) (*DexPriceAggregatorUniswapV3, error) {
	dexPriceAggregatorUniswapV3 := NewDexPriceAggregatorUniswapV3()
	address := poolState.DexPriceAggregatorAddress.String()

	if err := r.readData(ctx, address, dexPriceAggregatorUniswapV3); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read data")
		return nil, err
	}

	if err := r.readOverriddenPoolForRoute(ctx, address, dexPriceAggregatorUniswapV3, poolState.SystemSettings.AtomicEquivalentForDexPricing); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read overridden pool for route")
		return nil, err
	}

	if err := r.readPoolData(ctx, dexPriceAggregatorUniswapV3, poolState.SystemSettings.AtomicEquivalentForDexPricing); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read pool data")
		return nil, err
	}

	if err := r.readPoolObservationsData(ctx, dexPriceAggregatorUniswapV3); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read pool observations data")
		return nil, err
	}

	if err := r.readPoolTickCumulativeData(ctx, dexPriceAggregatorUniswapV3, poolState.SystemSettings.AtomicTwapWindow); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read pool tick cumulative data")
		return nil, err
	}

	return dexPriceAggregatorUniswapV3, nil
}

// readData reads data which required no parameters, included:
// - DefaultPoolFee
// - UniswapV3Factory
// - Weth
func (r *DexPriceAggregatorUniswapV3Reader) readData(
	ctx context.Context,
	address string,
	dexPriceAggregator *DexPriceAggregatorUniswapV3,
) error {
	req := r.ethrpcClient.
		NewRequest().
		SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: DexPriceAggregatorUniswapV3MethodDefaultPoolFee,
			Params: nil,
		}, []interface{}{&dexPriceAggregator.DefaultPoolFee}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: DexPriceAggregatorUniswapV3MethodUniswapV3Factory,
			Params: nil,
		}, []interface{}{&dexPriceAggregator.UniswapV3Factory}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: DexPriceAggregatorUniswapV3MethodWeth,
			Params: nil,
		}, []interface{}{&dexPriceAggregator.Weth})

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read data")
		return err
	}

	return nil
}

func (r *DexPriceAggregatorUniswapV3Reader) readOverriddenPoolForRoute(
	ctx context.Context,
	address string,
	dexPriceAggregator *DexPriceAggregatorUniswapV3,
	atomicEquivalentForDexPricing map[string]Token,
) error {
	tokens := make([]Token, 0, len(atomicEquivalentForDexPricing))
	for _, token := range atomicEquivalentForDexPricing {
		tokens = append(tokens, token)
	}
	tokensLen := len(tokens)

	var routeFromPoolKeys []string
	for i := 0; i < tokensLen; i++ {
		for j := i + 1; j < tokensLen; j++ {
			poolKey := getPoolKey(
				tokens[i].Address,
				tokens[j].Address,
				dexPriceAggregator.DefaultPoolFee,
			)

			routeFromPoolKey := _identifyRouteFromPoolKey(poolKey)
			routeFromPoolKeys = append(routeFromPoolKeys, routeFromPoolKey)
		}
	}

	overriddenPoolForRoutes := make([]common.Address, len(routeFromPoolKeys))

	req := r.ethrpcClient.NewRequest().SetContext(ctx)

	for i, routeFromPoolKey := range routeFromPoolKeys {
		routeFromPoolKeyBytes := eth.StringToBytes32(routeFromPoolKey)

		req.AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: DexPriceAggregatorUniswapV3MethodOverriddenPoolForRoute,
			Params: []interface{}{routeFromPoolKeyBytes},
		}, []interface{}{&overriddenPoolForRoutes[i]})
	}

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read overridden pool for route")
		return err
	}

	for i, routeFromPoolKey := range routeFromPoolKeys {
		dexPriceAggregator.OverriddenPoolForRoute[routeFromPoolKey] = overriddenPoolForRoutes[i]
	}

	return nil
}

// readPoolData reads pool data which required no parameter, included:
// - UniswapV3Slot0
func (r *DexPriceAggregatorUniswapV3Reader) readPoolData(
	ctx context.Context,
	dexPriceAggregator *DexPriceAggregatorUniswapV3,
	atomicEquivalentForDexPricing map[string]Token,
) error {
	tokensArr := make([]Token, 0, len(atomicEquivalentForDexPricing))
	for _, token := range atomicEquivalentForDexPricing {
		tokensArr = append(tokensArr, token)
	}

	poolAddresses := getPoolCombinationsFromTokens(dexPriceAggregator, tokensArr)
	poolsLen := len(poolAddresses)
	poolSlot0s := make([]Slot0, poolsLen)

	req := r.ethrpcClient.NewRequest().SetContext(ctx)
	for i, pool := range poolAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    uniswapV3Pool,
			Target: pool.String(),
			Method: UniswapV3PoolMethodSlot0,
			Params: nil,
		}, []interface{}{&poolSlot0s[i]})
	}

	_, err := req.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("dex price aggregator uniswapV3 reader reads pool data error")
		return err
	}

	for i, poolAddress := range poolAddresses {
		if poolSlot0s[i].Tick == nil {
			continue
		}

		dexPriceAggregator.UniswapV3Slot0[poolAddress.String()] = poolSlot0s[i]
	}

	return nil
}

// readPoolObservationsData reads UniswapV3 pool observation data, included:
// - UniswapV3Observations
func (r *DexPriceAggregatorUniswapV3Reader) readPoolObservationsData(
	ctx context.Context,
	dexPriceAggregator *DexPriceAggregatorUniswapV3,
) error {
	uniswapV3Slot0 := dexPriceAggregator.UniswapV3Slot0
	poolsLen := len(uniswapV3Slot0)
	poolAddresses := make([]string, 0, poolsLen)

	for poolAddress := range uniswapV3Slot0 {
		poolAddresses = append(poolAddresses, poolAddress)
	}

	observations := make([]OracleObservation, poolsLen)
	prevObservations := make([]OracleObservation, poolsLen)

	req := r.ethrpcClient.NewRequest().SetContext(ctx)
	for i, poolAddress := range poolAddresses {
		observationIndex := uniswapV3Slot0[poolAddress].ObservationIndex
		observationCardinality := uniswapV3Slot0[poolAddress].ObservationCardinality
		prevIndex := (observationIndex + observationCardinality - 1) % observationCardinality

		req.
			AddCall(&ethrpc.Call{
				ABI:    uniswapV3Pool,
				Target: poolAddress,
				Method: UniswapV3PoolMethodObservations,
				Params: []interface{}{big.NewInt(int64(observationIndex))},
			}, []interface{}{&observations[i]}).
			AddCall(&ethrpc.Call{
				ABI:    uniswapV3Pool,
				Target: poolAddress,
				Method: UniswapV3PoolMethodObservations,
				Params: []interface{}{big.NewInt(int64(prevIndex))},
			}, []interface{}{&prevObservations[i]})
	}

	_, err := req.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("dex price aggregator uniswapV3 reader read pool observations data error")
		return err
	}

	for i, poolAddress := range poolAddresses {
		observationIndex := uniswapV3Slot0[poolAddress].ObservationIndex
		observationCardinality := uniswapV3Slot0[poolAddress].ObservationCardinality

		prevIndex := (observationIndex + observationCardinality - 1) % observationCardinality

		dexPriceAggregator.UniswapV3Observations[poolAddress] = map[uint16]OracleObservation{
			observationIndex: observations[i],
			prevIndex:        prevObservations[i],
		}
	}

	return nil
}

// readPoolTickCumulativeData reads UniswapV3 pool cumulative data, included:
// - TickCumulatives
func (r *DexPriceAggregatorUniswapV3Reader) readPoolTickCumulativeData(
	ctx context.Context,
	dexPriceAggregator *DexPriceAggregatorUniswapV3,
	atomicTwapWindow *big.Int,
) error {
	uniswapV3Slot0 := dexPriceAggregator.UniswapV3Slot0
	poolsLen := len(uniswapV3Slot0)
	poolAddresses := make([]string, 0, poolsLen)

	for poolAddress := range uniswapV3Slot0 {
		poolAddresses = append(poolAddresses, poolAddress)
	}

	type ObserveResult struct {
		TickCumulatives                    []*big.Int `json:"TickCumulatives"`
		SecondsPerLiquidityCumulativeX128s []*big.Int `json:"SecondsPerLiquidityCumulativeX128s"`
	}

	observeResult := make([]ObserveResult, poolsLen)

	req := r.ethrpcClient.NewRequest().SetContext(ctx)
	for i, poolAddress := range poolAddresses {
		secondAgos := make([]uint32, 2)
		secondAgos[0] = uint32(atomicTwapWindow.Int64())
		secondAgos[1] = 0

		req.AddCall(&ethrpc.Call{
			ABI:    uniswapV3Pool,
			Target: poolAddress,
			Method: UniswapV3PoolMethodObserve,
			Params: []interface{}{secondAgos},
		}, []interface{}{&observeResult[i]})
	}

	_, err := req.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("dex price aggregator uniswapV3 reader reads pool tick cummulative data error")
		return err
	}

	for i, poolAddress := range poolAddresses {
		if len(observeResult[i].TickCumulatives) != 2 {
			continue
		}

		dexPriceAggregator.TickCumulatives[poolAddress] = observeResult[i].TickCumulatives
	}

	return nil
}

func getPoolCombinationsFromTokens(
	dexPriceAggregator *DexPriceAggregatorUniswapV3,
	tokens []Token,
) []common.Address {
	var pools []common.Address
	tokensLen := len(tokens)

	for i := 0; i < tokensLen; i++ {
		for j := i + 1; j < tokensLen; j++ {
			poolKey := getPoolKey(tokens[i].Address, tokens[j].Address, dexPriceAggregator.DefaultPoolFee)
			pool, err := _getPoolForRoute(dexPriceAggregator, poolKey)

			if err != nil || eth.IsZeroAddress(pool) {
				continue
			}

			pools = append(pools, pool)
		}
	}

	return pools
}

// @notice Fetch the Uniswap V3 pool to be queried for a route denoted by a PoolKey
// @param _poolKey PoolKey representing the route
// @return pool Address of the Uniswap V3 pool to use for the route
func _getPoolForRoute(
	dexPriceAggregator *DexPriceAggregatorUniswapV3,
	_poolKey PoolKey,
) (common.Address, error) {
	pool := _getOverriddenPool(dexPriceAggregator, _poolKey)
	if !eth.IsZeroAddress(pool) {
		return pool, nil
	}

	pool, err := computeAddress(dexPriceAggregator.UniswapV3Factory, _poolKey)
	if err != nil {
		return common.Address{}, err
	}

	return pool, nil
}

// @notice Fetch an overridden pool for a route denoted by a PoolKey, if any
// @param _poolKey PoolKey representing the route
// @return pool Address of the Uniswap V3 pool overridden for the route.
//
//	address(0) if no overridden pool has been set.
func _getOverriddenPool(dexPriceAggregator *DexPriceAggregatorUniswapV3, _poolKey PoolKey) common.Address {
	return dexPriceAggregator.OverriddenPoolForRoute[_identifyRouteFromPoolKey(_poolKey)]
}
