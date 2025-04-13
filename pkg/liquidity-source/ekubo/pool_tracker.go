package ekubo

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kutils/klog"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	bignum "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := klog.WithFields(ctx, klog.Fields{
		"dexId":       d.config.DexId,
		"poolAddress": p.Address,
	})
	defer func() {
		lg.Info("Finish updating state.")
	}()

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, err
	}

	if err := d.applyLogs(&p, params.Logs, staticExtra.PoolKey, &extra.PoolState); err != nil {
		lg.Errorf("log application failed, falling back to RPC, error: %v", err)
		extra.PoolState, err = d.forceUpdateState(ctx, staticExtra.PoolKey)
		if err != nil {
			return p, err
		}
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	balances, err := d.calcBalances(&extra.PoolState)
	if err != nil {
		return p, err
	}

	p.Reserves = lo.Map(balances, func(v big.Int, _ int) string { return v.String() })
	p.Timestamp = time.Now().Unix()
	p.Extra = string(extraBytes)
	p.BlockNumber = extra.PoolState.GetBlockNumber()

	return p, nil
}

func (d *PoolTracker) calcBalances(state *quoting.PoolState) ([]big.Int, error) {
	stateSqrtRatio := new(big.Int).Set(state.SqrtRatio)

	balances := make([]big.Int, 2)
	liquidity := new(big.Int)
	sqrtRatio := new(big.Int).Set(math.MinSqrtRatio)

	for _, tick := range state.Ticks {
		tickSqrtRatio := math.ToSqrtRatio(tick.Number)
		var minAmount1SqrtRatio, maxAmount0SqrtRatio *big.Int
		if stateSqrtRatio.Cmp(tickSqrtRatio) > 0 {
			minAmount1SqrtRatio = new(big.Int).Set(tickSqrtRatio)
		} else {
			minAmount1SqrtRatio = new(big.Int).Set(stateSqrtRatio)
		}

		if stateSqrtRatio.Cmp(sqrtRatio) > 0 {
			maxAmount0SqrtRatio = new(big.Int).Set(stateSqrtRatio)
		} else {
			maxAmount0SqrtRatio = new(big.Int).Set(sqrtRatio)
		}

		if sqrtRatio.Cmp(minAmount1SqrtRatio) < 0 {
			amount1Delta, err := math.Amount1Delta(sqrtRatio, minAmount1SqrtRatio, liquidity, false)
			if err != nil {
				return nil, math.ErrAmount1DeltaOverflow
			}
			balances[1].Add(&balances[1], amount1Delta)
		}
		if maxAmount0SqrtRatio.Cmp(tickSqrtRatio) < 0 {
			amount0Delta, err := math.Amount0Delta(maxAmount0SqrtRatio, tickSqrtRatio, liquidity, false)
			if err != nil {
				return nil, math.ErrAmount0DeltaOverflow
			}
			balances[0].Add(&balances[0], amount0Delta)
		}

		sqrtRatio.Set(tickSqrtRatio)
		liquidity.Add(liquidity, tick.LiquidityDelta)
	}

	return balances, nil
}

func (d *PoolTracker) applyLogs(p *entity.Pool, logs []types.Log, poolKey *quoting.PoolKey, poolState *quoting.PoolState) error {
	for _, log := range logs {
		if !strings.EqualFold(d.config.Core, log.Address.String()) {
			continue
		}

		if log.Removed {
			continue
		}

		if len(log.Topics) == 0 {
			if err := handleSwappedEvent(log.Data, poolKey, poolState); err != nil {
				return fmt.Errorf("handling swap event: %w", err)
			}
		} else if log.Topics[0] == positionUpdatedEvent.ID {
			if err := handlePositionUpdatedEvent(log.Data, poolKey, poolState); err != nil {
				return fmt.Errorf("handling position updated event: %w", err)
			}
		}

		poolState.SetBlockNumber(log.BlockNumber)
	}

	return nil
}

func (d *PoolTracker) forceUpdateState(ctx context.Context, poolKey *quoting.PoolKey) (quoting.PoolState, error) {
	logger.WithFields(logger.Fields{
		"dexId":       d.config.DexId,
		"poolAddress": poolKey.StringId(),
	}).Info("update state from data fetcher")

	poolStates, err := fetchPoolStates(
		ctx,
		d.ethrpcClient,
		d.config.DataFetcher,
		[]*quoting.PoolKey{poolKey})
	if err != nil {
		return quoting.PoolState{}, fmt.Errorf("fetching pool state: %w", err)
	}

	return poolStates[0], nil
}

func handleSwappedEvent(data []byte, poolKey *quoting.PoolKey, poolState *quoting.PoolState) error {
	n := new(big.Int).SetBytes(data)

	poolId := new(big.Int).And(new(big.Int).Rsh(n, 512), bignum.MAX_UINT_256)

	expectedPoolId, err := poolKey.NumId()
	if err != nil {
		return fmt.Errorf("computing expected pool id: %w", err)
	}
	if expectedPoolId.Cmp(poolId) != 0 {
		return nil
	}

	tickRaw := new(big.Int).And(n, math.U32Max)
	if tickRaw.Cmp(math.TwoPow31) < 0 {
		poolState.ActiveTick = int32(tickRaw.Uint64())
	} else {
		poolState.ActiveTick = int32(tickRaw.Sub(tickRaw, math.TwoPow32).Int64())
	}
	n.Rsh(n, 32)

	sqrtRatioAfterCompact := new(big.Int).And(n, math.U96Max)
	n.Rsh(n, 96)

	poolState.SqrtRatio = math.FloatSqrtRatioToFixed(sqrtRatioAfterCompact)
	poolState.Liquidity.And(n, bignum.MAX_UINT_128)

	return nil
}

func handlePositionUpdatedEvent(data []byte, poolKey *quoting.PoolKey, poolState *quoting.PoolState) error {
	values, err := positionUpdatedEvent.Inputs.Unpack(data)
	if err != nil {
		return fmt.Errorf("unpacking event data: %w", err)
	}

	poolId, ok := values[1].([32]byte)
	if !ok {
		return errors.New(`failed to parse "poolId"`)
	}

	params, ok := values[2].(struct {
		Salt   [32]uint8 `json:"salt"`
		Bounds struct {
			Lower int32 `json:"lower"`
			Upper int32 `json:"upper"`
		} `json:"bounds"`
		LiquidityDelta *big.Int `json:"liquidityDelta"`
	})
	if !ok {
		return errors.New(`failed to parse "params"`)
	}

	liquidityDelta, lowerBound, upperBound := params.LiquidityDelta, params.Bounds.Lower, params.Bounds.Upper

	if liquidityDelta.Sign() == 0 {
		return nil
	}

	expectedPoolId, err := poolKey.NumId()
	if err != nil {
		return fmt.Errorf("computing expected pool id: %w", err)
	}

	if expectedPoolId.Cmp(new(big.Int).SetBytes(poolId[:])) != 0 {
		return nil
	}

	poolState.UpdateTick(lowerBound, liquidityDelta, false, false)
	poolState.UpdateTick(upperBound, liquidityDelta, true, false)

	if poolState.ActiveTick >= lowerBound && poolState.ActiveTick < upperBound {
		poolState.Liquidity.Add(poolState.Liquidity, liquidityDelta)
	}

	return nil
}
