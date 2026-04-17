package ambient

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
	stateTracker *StateTracker
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
		stateTracker: NewStateTracker(ethrpcClient.GetETHClient(), common.HexToAddress(cfg.SwapDexContractAddress)),
	}, nil
}

func (t *PoolTracker) tickWindow(prevCurve *CurveState) TickWindow {
	if t.cfg.TickRange <= 0 {
		return FullTickWindow
	}
	var center int32
	if prevCurve != nil && prevCurve.PriceRoot != nil && prevCurve.PriceRoot.Sign() > 0 {
		center = GetTickAtSqrtRatio(prevCurve.PriceRoot)
	}
	minTick := center - t.cfg.TickRange
	maxTick := center + t.cfg.TickRange
	if minTick < FullTickWindow.MinTick {
		minTick = FullTickWindow.MinTick
	}
	if maxTick > FullTickWindow.MaxTick {
		maxTick = FullTickWindow.MaxTick
	}
	return TickWindow{MinTick: minTick, MaxTick: maxTick}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")
	defer func() {
		logger.WithFields(logger.Fields{
			"pool_id":     p.Address,
			"duration_ms": time.Since(startTime).Milliseconds(),
		}).Info("Finished getting new pool state")
	}()

	var (
		poolAddr           = common.HexToAddress(p.Address)
		nativeTokenAddress = common.HexToAddress(t.cfg.NativeTokenAddress)

		extra Extra
	)

	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		logger.WithFields(logger.Fields{"dex_id": t.cfg.DexID, "address": p.Address, "err": err}).
			Error("could not json.Unmarshal Extra")
		return p, fmt.Errorf("could not json.Unmarshal Extra: %w", err)
	}
	if len(extra.TokenPairs) == 0 {
		return p, ErrNoTrackedPairs
	}

	blockNumber, err := t.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		return p, fmt.Errorf("get block number: %w", err)
	}
	blockNumBI := new(big.Int).SetUint64(blockNumber)

	reserves := make([]*big.Int, len(p.Tokens))
	rpcRequest := t.ethrpcClient.NewRequest()
	rpcRequest.SetContext(ctx)
	rpcRequest.SetBlockNumber(blockNumBI)

	for i, token := range p.Tokens {
		tokenAddr := common.HexToAddress(token.Address)
		if tokenAddr == nativeTokenAddress {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    multicallABI,
				Target: t.cfg.MulticallContractAddress,
				Method: "getEthBalance",
				Params: []any{poolAddr},
			}, []any{&reserves[i]})
		} else {
			rpcRequest.AddCall(&ethrpc.Call{
				ABI:    erc20ABI,
				Target: tokenAddr.Hex(),
				Method: "balanceOf",
				Params: []any{poolAddr},
			}, []any{&reserves[i]})
		}
	}

	if _, err := rpcRequest.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{"poolAddress": p.Address, "error": err}).
			Error("failed to call multicall contract TryAggregate")
		return p, err
	}

	for pair, pairInfo := range extra.TokenPairs {
		if pairInfo == nil {
			pairInfo = &TokenPairInfo{PoolIdx: new(big.Int).Set(t.cfg.PoolIdx)}
			extra.TokenPairs[pair] = pairInfo
		}
		if pairInfo.PoolIdx == nil || pairInfo.PoolIdx.Sign() == 0 {
			pairInfo.PoolIdx = new(big.Int).Set(t.cfg.PoolIdx)
		}

		poolIdx := pairInfo.PoolIdx.Uint64()
		var state *TrackerExtra
		if pairInfo.State == nil {
			state, err = t.stateTracker.LoadWindow(
				ctx, pair.Base, pair.Quote, poolIdx, blockNumBI,
				t.tickWindow(nil),
			)
		} else {
			var changed bool
			state, changed, err = t.stateTracker.Refresh(ctx, pairInfo.State, blockNumBI)
			if err == nil && !changed {
				state = pairInfo.State
			}
		}
		if err != nil {
			return p, fmt.Errorf("track pair %s: %w", pair, err)
		}
		pairInfo.State = state
	}

	encodedExtra, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{"poolAddress": p.Address, "error": err}).
			Error("failed to marshal extra data")
		return p, err
	}

	p.Extra = string(encodedExtra)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber
	for i := len(p.Reserves); i < len(p.Tokens); i++ {
		p.Reserves = append(p.Reserves, "")
	}
	for i, reserve := range reserves {
		if reserve == nil {
			p.Reserves[i] = "0"
			continue
		}
		p.Reserves[i] = reserve.String()
	}

	return p, nil
}
