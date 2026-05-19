package unipool

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

// rpcState bundles the per-pool storage values we need from the pair.
//
// Why these and not previewReserves: we store the raw (un-interpolated) snapshot at
// lastUpdateTimestamp so the off-chain simulator can replay the virtual-reserve
// decay in Go at quote time, matching previewVirtualReservesElapsed.
type rpcState struct {
	reserves              reservesABI
	virtualReserves       virtualReservesABI
	lastUpdateTimestamp   *big.Int
	priceDecay            *big.Int
	fees                  feesBpsABI
	totalBorrowed0        *big.Int
	totalBorrowed1        *big.Int
	swapPriceToleranceBps uint16
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	start := time.Now()
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	state, blockNumber, err := t.fetchState(ctx, p.Address)
	if err != nil {
		return p, err
	}
	if p.BlockNumber > blockNumber {
		logger.WithFields(logger.Fields{
			"pool_id":           p.Address,
			"pool_block_number": p.BlockNumber,
			"data_block_number": blockNumber,
		}).Info("skip update: data block number is less than current pool block number")
		return p, nil
	}

	extra := Extra{
		Reserve0:              state.reserves.Reserve0,
		Reserve1:              state.reserves.Reserve1,
		VirtualReserve0In:     state.virtualReserves.VirtualReserve0In,
		VirtualReserve0Out:    state.virtualReserves.VirtualReserve0Out,
		VirtualReserve1In:     state.virtualReserves.VirtualReserve1In,
		VirtualReserve1Out:    state.virtualReserves.VirtualReserve1Out,
		LastUpdateTimestamp:   state.lastUpdateTimestamp.Uint64(),
		PriceDecay:            state.priceDecay.Uint64(),
		FeeLpBps:              state.fees.FeeLpBps,
		FeePoolBps:            state.fees.FeePoolBps,
		TotalBorrowed0:        state.totalBorrowed0,
		TotalBorrowed1:        state.totalBorrowed1,
		SwapPriceToleranceBps: state.swapPriceToleranceBps,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{
		state.reserves.Reserve0.String(),
		state.reserves.Reserve1.String(),
	}
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	logger.WithFields(logger.Fields{
		"pool_id":     p.Address,
		"block":       blockNumber,
		"duration_ms": time.Since(start).Milliseconds(),
	}).Info("Finished getting new pool state")

	return p, nil
}

func (t *PoolTracker) fetchState(ctx context.Context, pairAddress string) (*rpcState, uint64, error) {
	state := &rpcState{
		reserves:            reservesABI{Reserve0: new(big.Int), Reserve1: new(big.Int)},
		virtualReserves:     virtualReservesABI{},
		lastUpdateTimestamp: new(big.Int),
		priceDecay:          new(big.Int),
		totalBorrowed0:      new(big.Int),
		totalBorrowed1:      new(big.Int),
	}

	// getVirtualReserves returns a single tuple. go-ethereum's abi.copyAtomic
	// (accounts/abi/argument.go:131-138) takes `dst.Field(0)` when the receiver
	// is a struct, so the tuple destination must be wrapped in a parent struct
	// with one field of the target type. Without this wrapper, the unpacker
	// silently fails with "cannot unmarshal struct in to *big.Int" and the VR
	// fields stay nil.
	var vrRecv struct{ Reserves virtualReservesABI }

	req := t.ethrpcClient.NewRequest().SetContext(ctx)

	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pairAddress, Method: pairMethodGetReserves},
		[]any{&state.reserves})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pairAddress, Method: pairMethodGetVirtualReserves},
		[]any{&vrRecv})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pairAddress, Method: pairMethodGetLastUpdateTimestamp},
		[]any{&state.lastUpdateTimestamp})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pairAddress, Method: pairMethodGetPriceDecay},
		[]any{&state.priceDecay})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pairAddress, Method: pairMethodGetFeesBps},
		[]any{&state.fees})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pairAddress, Method: pairMethodGetTotalBorrowed0},
		[]any{&state.totalBorrowed0})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pairAddress, Method: pairMethodGetTotalBorrowed1},
		[]any{&state.totalBorrowed1})
	req.AddCall(&ethrpc.Call{ABI: uniPoolPairABI, Target: pairAddress, Method: pairMethodGetSwapPriceToleranceBps},
		[]any{&state.swapPriceToleranceBps})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, 0, err
	}
	state.virtualReserves = vrRecv.Reserves
	return state, resp.BlockNumber.Uint64(), nil
}
