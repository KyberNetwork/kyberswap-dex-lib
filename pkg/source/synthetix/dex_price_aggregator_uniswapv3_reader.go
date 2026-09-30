package synthetix

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

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

	if err := r.readData(ctx, address, dexPriceAggregatorUniswapV3, poolState.BlockNumber); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read data")
		return nil, err
	}

	if err := r.readOverriddenPoolForRoute(ctx, address, dexPriceAggregatorUniswapV3, poolState.SystemSettings.AtomicEquivalentForDexPricing, poolState.BlockNumber); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read overridden pool for route")
		return nil, err
	}

	if err := r.readPoolData(ctx, dexPriceAggregatorUniswapV3, poolState.SystemSettings.AtomicEquivalentForDexPricing, poolState.BlockNumber); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read pool data")
		return nil, err
	}

	if err := r.readPoolObservationsData(ctx, dexPriceAggregatorUniswapV3, poolState.BlockNumber); err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read pool observations data")
		return nil, err
	}

	if err := r.readPoolTickCumulativeData(ctx, dexPriceAggregatorUniswapV3, poolState.SystemSettings.AtomicTwapWindow, poolState.BlockNumber); err != nil {
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
	blockNumber uint64,
) error {
	var defaultPoolFee *uint256.Int

	req := newRequest(r.ethrpcClient, ctx, blockNumber).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: DexPriceAggregatorUniswapV3MethodDefaultPoolFee,
			Params: nil,
		}, []any{&defaultPoolFee}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: DexPriceAggregatorUniswapV3MethodUniswapV3Factory,
			Params: nil,
		}, []any{&dexPriceAggregator.UniswapV3Factory}).
		AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: DexPriceAggregatorUniswapV3MethodWeth,
			Params: nil,
		}, []any{&dexPriceAggregator.Weth})

	_, err := req.Aggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("can not read data")
		return err
	}

	dexPriceAggregator.DefaultPoolFee = u256ToBig(defaultPoolFee)

	return nil
}

func (r *DexPriceAggregatorUniswapV3Reader) readOverriddenPoolForRoute(
	ctx context.Context,
	address string,
	dexPriceAggregator *DexPriceAggregatorUniswapV3,
	atomicEquivalentForDexPricing map[string]Token,
	blockNumber uint64,
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

	req := newRequest(r.ethrpcClient, ctx, blockNumber)

	for i, routeFromPoolKey := range routeFromPoolKeys {
		routeFromPoolKeyBytes := eth.StringToBytes32(routeFromPoolKey)

		req.AddCall(&ethrpc.Call{
			ABI:    r.abi,
			Target: address,
			Method: DexPriceAggregatorUniswapV3MethodOverriddenPoolForRoute,
			Params: []any{routeFromPoolKeyBytes},
		}, []any{&overriddenPoolForRoutes[i]})
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
	blockNumber uint64,
) error {
	type slot0RPC struct {
		SqrtPriceX96               *uint256.Int `json:"sqrtPriceX96"`
		Tick                       *big.Int     `json:"tick"`
		ObservationIndex           uint16       `json:"observationIndex"`
		ObservationCardinality     uint16       `json:"observationCardinality"`
		ObservationCardinalityNext uint16       `json:"observationCardinalityNext"`
		FeeProtocol                uint8        `json:"feeProtocol"`
		Unlocked                   bool         `json:"unlocked"`
	}

	tokensArr := make([]Token, 0, len(atomicEquivalentForDexPricing))
	for _, token := range atomicEquivalentForDexPricing {
		tokensArr = append(tokensArr, token)
	}

	poolAddresses := getPoolCombinationsFromTokens(dexPriceAggregator, tokensArr)
	poolsLen := len(poolAddresses)
	poolSlot0s := make([]slot0RPC, poolsLen)

	req := newRequest(r.ethrpcClient, ctx, blockNumber)
	for i, pool := range poolAddresses {
		req.AddCall(&ethrpc.Call{
			ABI:    uniswapV3Pool,
			Target: pool.String(),
			Method: UniswapV3PoolMethodSlot0,
			Params: nil,
		}, []any{&poolSlot0s[i]})
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

		dexPriceAggregator.UniswapV3Slot0[poolAddress.String()] = Slot0{
			SqrtPriceX96:               u256ToBig(poolSlot0s[i].SqrtPriceX96),
			Tick:                       poolSlot0s[i].Tick,
			ObservationIndex:           poolSlot0s[i].ObservationIndex,
			ObservationCardinality:     poolSlot0s[i].ObservationCardinality,
			ObservationCardinalityNext: poolSlot0s[i].ObservationCardinalityNext,
			FeeProtocol:                poolSlot0s[i].FeeProtocol,
			Unlocked:                   poolSlot0s[i].Unlocked,
		}
	}

	return nil
}

// readPoolObservationsData reads UniswapV3 pool observation data, included:
// - UniswapV3Observations
func (r *DexPriceAggregatorUniswapV3Reader) readPoolObservationsData(
	ctx context.Context,
	dexPriceAggregator *DexPriceAggregatorUniswapV3,
	blockNumber uint64,
) error {
	type oracleObservationRPC struct {
		BlockTimestamp                    uint32       `json:"blockTimestamp"`
		TickCumulative                    *big.Int     `json:"tickCumulative"`
		SecondsPerLiquidityCumulativeX128 *uint256.Int `json:"secondsPerLiquidityCumulativeX128"`
		Initialized                       bool         `json:"initialized"`
	}

	uniswapV3Slot0 := dexPriceAggregator.UniswapV3Slot0
	poolsLen := len(uniswapV3Slot0)
	poolAddresses := make([]string, 0, poolsLen)

	for poolAddress := range uniswapV3Slot0 {
		poolAddresses = append(poolAddresses, poolAddress)
	}

	observations := make([]oracleObservationRPC, poolsLen)
	prevObservations := make([]oracleObservationRPC, poolsLen)

	req := newRequest(r.ethrpcClient, ctx, blockNumber)
	for i, poolAddress := range poolAddresses {
		observationIndex := uniswapV3Slot0[poolAddress].ObservationIndex
		observationCardinality := uniswapV3Slot0[poolAddress].ObservationCardinality
		prevIndex := (observationIndex + observationCardinality - 1) % observationCardinality

		req.
			AddCall(&ethrpc.Call{
				ABI:    uniswapV3Pool,
				Target: poolAddress,
				Method: UniswapV3PoolMethodObservations,
				Params: []any{big.NewInt(int64(observationIndex))},
			}, []any{&observations[i]}).
			AddCall(&ethrpc.Call{
				ABI:    uniswapV3Pool,
				Target: poolAddress,
				Method: UniswapV3PoolMethodObservations,
				Params: []any{big.NewInt(int64(prevIndex))},
			}, []any{&prevObservations[i]})
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
			observationIndex: {
				BlockTimestamp:                    observations[i].BlockTimestamp,
				TickCumulative:                    observations[i].TickCumulative,
				SecondsPerLiquidityCumulativeX128: u256ToBig(observations[i].SecondsPerLiquidityCumulativeX128),
				Initialized:                       observations[i].Initialized,
			},
			prevIndex: {
				BlockTimestamp:                    prevObservations[i].BlockTimestamp,
				TickCumulative:                    prevObservations[i].TickCumulative,
				SecondsPerLiquidityCumulativeX128: u256ToBig(prevObservations[i].SecondsPerLiquidityCumulativeX128),
				Initialized:                       prevObservations[i].Initialized,
			},
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
	blockNumber uint64,
) error {
	uniswapV3Slot0 := dexPriceAggregator.UniswapV3Slot0
	poolsLen := len(uniswapV3Slot0)
	poolAddresses := make([]string, 0, poolsLen)

	for poolAddress := range uniswapV3Slot0 {
		poolAddresses = append(poolAddresses, poolAddress)
	}

	type ObserveResult struct {
		TickCumulatives                    []*big.Int     `json:"TickCumulatives"`
		SecondsPerLiquidityCumulativeX128s []*uint256.Int `json:"SecondsPerLiquidityCumulativeX128s"`
	}

	observeResult := make([]ObserveResult, poolsLen)

	req := newRequest(r.ethrpcClient, ctx, blockNumber)
	for i, poolAddress := range poolAddresses {
		secondAgos := make([]uint32, 2)
		secondAgos[0] = uint32(atomicTwapWindow.Int64())
		secondAgos[1] = 0

		req.AddCall(&ethrpc.Call{
			ABI:    uniswapV3Pool,
			Target: poolAddress,
			Method: UniswapV3PoolMethodObserve,
			Params: []any{secondAgos},
		}, []any{&observeResult[i]})
	}

	_, err := req.TryAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": r.cfg.DexID,
			"error": err,
		}).Error("dex price aggregator uniswapV3 reader reads pool tick cumulative data error")
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
