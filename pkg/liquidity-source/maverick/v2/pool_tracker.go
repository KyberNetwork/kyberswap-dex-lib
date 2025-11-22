package maverickv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type (
	PoolTracker struct {
		config       *Config
		ethrpcClient *ethrpc.Client
	}

	GetStateResult struct {
		ReserveA           *big.Int `json:"reserveA"`
		ReserveB           *big.Int `json:"reserveB"`
		LastTwaD8          int64    `json:"lastTwaD8"`
		LastLogPriceD8     int64    `json:"lastLogPriceD8"`
		LastTimestamp      *big.Int `json:"lastTimestamp"`
		ActiveTick         int32    `json:"activeTick"`
		IsLocked           bool     `json:"isLocked"`
		BinCounter         uint32   `json:"binCounter"`
		ProtocolFeeRatioD3 uint8    `json:"protocolFeeRatioD3"`
		FeeAIn             uint64   `json:"feeAIn"` // Fee for tokenA -> tokenB swaps
		FeeBIn             uint64   `json:"feeBIn"` // Fee for tokenB -> tokenA swaps
	}

	// because the result is a tuple with internal type = struct IMaverickV2Pool.State, we need to wrap it in a struct like this
	GetStateResultWrapper struct {
		GetStateResult
	}
)

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	startTime := time.Now()

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	state, blockNumber, err := t.getState(ctx, p.Address, overrides)
	if err != nil {
		return p, err
	}

	if p.BlockNumber > blockNumber.Uint64() {
		logger.
			WithFields(
				logger.Fields{
					"pool_id":           p.Address,
					"pool_block_number": p.BlockNumber,
					"data_block_number": blockNumber.Uint64(),
				},
			).
			Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	logger.WithFields(
		logger.Fields{
			"pool_id":          p.Address,
			"old_reserve":      p.Reserves,
			"new_reserve":      state,
			"old_block_number": p.BlockNumber,
			"new_block_number": blockNumber,
			"duration_ms":      time.Since(startTime).Milliseconds(),
		},
	).Info("Finished getting new pool state")

	return t.updatePool(p, state, blockNumber, overrides)
}

func (t *PoolTracker) getState(
	ctx context.Context,
	poolAddress string,
	overrides map[common.Address]gethclient.OverrideAccount,
) (State, *big.Int, error) {
	var getStateResult GetStateResultWrapper
	var feeAIn, feeBIn *big.Int

	getStateRequest := t.ethrpcClient.NewRequest().SetContext(ctx).SetRequireSuccess(true)
	if overrides != nil {
		getStateRequest.SetOverrides(overrides)
	}

	getStateRequest.AddCall(&ethrpc.Call{
		ABI:    maverickV2PoolABI,
		Target: poolAddress,
		Method: poolMethodGetState,
		Params: nil,
	}, []any{
		&getStateResult,
	})

	// Add calls to get current fees
	getStateRequest.AddCall(&ethrpc.Call{
		ABI:    maverickV2PoolABI,
		Target: poolAddress,
		Method: "fee",
		Params: []any{true}, // true for tokenAIn
	}, []any{&feeAIn})

	getStateRequest.AddCall(&ethrpc.Call{
		ABI:    maverickV2PoolABI,
		Target: poolAddress,
		Method: "fee",
		Params: []any{false}, // false for tokenBIn
	}, []any{&feeBIn})

	resp, err := getStateRequest.TryBlockAndAggregate()
	if err != nil {
		return State{}, nil, err
	}

	return State{
		ReserveA:           getStateResult.ReserveA,
		ReserveB:           getStateResult.ReserveB,
		LastTimestamp:      getStateResult.LastTimestamp.Int64(),
		LastTwaD8:          getStateResult.LastTwaD8,
		LastLogPriceD8:     getStateResult.LastLogPriceD8,
		ActiveTick:         getStateResult.ActiveTick,
		IsLocked:           getStateResult.IsLocked,
		BinCounter:         getStateResult.BinCounter,
		ProtocolFeeRatioD3: getStateResult.ProtocolFeeRatioD3,
		FeeAIn:             feeAIn.Uint64(),
		FeeBIn:             feeBIn.Uint64(),
	}, resp.BlockNumber, nil
}

func (t *PoolTracker) getFullPoolState(
	ctx context.Context,
	poolAddress string,
	binCounter uint32,
	overrides map[common.Address]gethclient.OverrideAccount,
) (map[uint32]Bin, map[int32]Tick, error) {
	// Calculate number of batches needed (5000 items per batch)
	batchSize := DefaultBinBatchSize
	numBatches := (int(binCounter) / batchSize) + 1

	// Prepare all calls for aggregation
	var allCalls []*ethrpc.Call
	var callResults []FullPoolStateWrapper

	for i := 0; i < numBatches; i++ {
		startIndex := i * batchSize
		endIndex := (i + 1) * batchSize

		call := &ethrpc.Call{
			ABI:    maverickV2PoolLensABI,
			Target: t.config.PoolLensAddress,
			Method: poolLensMethodGetFullPoolState,
			Params: []any{common.HexToAddress(poolAddress), uint32(startIndex), uint32(endIndex)},
		}
		// fmt.Println("call debug:", common.HexToAddress(poolAddress), uint32(startIndex), uint32(endIndex))
		allCalls = append(allCalls, call)
		callResults = append(callResults, FullPoolStateWrapper{})
	}

	// Execute all calls in aggregate
	request := t.ethrpcClient.NewRequest().SetContext(ctx).SetRequireSuccess(true)
	if overrides != nil {
		request.SetOverrides(overrides)
	}

	for i, call := range allCalls {
		request.AddCall(call, []any{&callResults[i]})
	}
	_, err := request.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, err
	}

	// Now map the aggregated results to our needed struct
	bins := make(map[uint32]Bin)
	ticks := make(map[int32]Tick)
	for batchIndex, wrapper := range callResults {
		fullPoolState := wrapper.PoolState
		startIndex := batchIndex * batchSize

		// Process the batch results
		for binIndex, binState := range fullPoolState.BinStateMapping {
			if binIndex == 0 {
				continue // skip index 0 as per TypeScript implementation
			}

			binId := binIndex + startIndex
			// Convert BinStateMapping to Bin
			bin := Bin{
				MergeBinBalance: uint256.MustFromBig(binState.MergeBinBalance),
				MergeId:         binState.MergeId,
				TotalSupply:     uint256.MustFromBig(binState.TotalSupply),
				Kind:            binState.Kind,
				Tick:            binState.Tick,
				TickBalance:     uint256.MustFromBig(binState.TickBalance),
			}

			tickState := fullPoolState.TickStateMapping[binIndex]
			bins[uint32(binId)] = bin
			ticks[bin.Tick] = Tick{
				ReserveA:     uint256.MustFromBig(tickState.ReserveA),
				ReserveB:     uint256.MustFromBig(tickState.ReserveB),
				TotalSupply:  uint256.MustFromBig(tickState.TotalSupply),
				BinIdsByTick: make(map[uint8]uint32),
			}
			for i, binId := range tickState.BinIdsByTick {
				ticks[bin.Tick].BinIdsByTick[uint8(i)] = binId
			}
		}
	}

	return bins, ticks, nil
}

func (t *PoolTracker) updatePool(
	pool entity.Pool,
	state State,
	blockNumber *big.Int,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	pool.Reserves = entity.PoolReserves{state.ReserveA.String(), state.ReserveB.String()}
	pool.BlockNumber = blockNumber.Uint64()
	pool.Timestamp = time.Now().Unix()

	// Parse StaticExtra to get tick spacing
	var staticExtra StaticExtra
	if pool.StaticExtra != "" {
		if err := json.Unmarshal([]byte(pool.StaticExtra), &staticExtra); err != nil {
			logger.WithFields(logger.Fields{
				"pool_address": pool.Address,
				"error":        err,
			}).Warn("Failed to unmarshal static extra data")
		}
	}

	// Fetch full pool state with bins and ticks data
	bins, ticks, err := t.getFullPoolState(context.Background(), pool.Address, state.BinCounter, overrides)
	if err != nil {
		logger.WithFields(logger.Fields{
			"pool_address": pool.Address,
			"error":        err,
		}).Warn("Failed to fetch full pool state, using empty bins")
		bins = make(map[uint32]Bin)
		ticks = make(map[int32]Tick)
	}

	// Update extra data with actual values from the state and fetched data
	extra := Extra{
		FeeAIn:           state.FeeAIn,
		FeeBIn:           state.FeeBIn,
		ProtocolFeeRatio: state.ProtocolFeeRatioD3,
		Bins:             bins,
		Ticks:            ticks,
		ActiveTick:       state.ActiveTick,
		LastTwaD8:        state.LastTwaD8,
		Timestamp:        state.LastTimestamp,
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return pool, err
	}

	pool.Extra = string(extraBytes)

	return pool, nil
}
