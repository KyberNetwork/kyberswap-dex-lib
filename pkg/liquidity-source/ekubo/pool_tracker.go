package ekubo

import (
	"context"
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/math"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting"
	ekubo_pool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ekubo/quoting/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	coreAddress  common.Address
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	coreAddress := common.HexToAddress(config.Core)

	return &PoolTracker{
		coreAddress,
		config,
		ethrpcClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
		return p, fmt.Errorf("unmarshalling extra: %w", err)
	}

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, fmt.Errorf("unmarshalling staticExtra: %w", err)
	}

	poolKey := staticExtra.PoolKey

	err := d.applyLogs(params.Logs, poolKey, &extra.State)
	if err == nil {
		extraJson, err := json.Marshal(extra)
		if err != nil {
			return p, fmt.Errorf("marshalling extra: %w", err)
		}

		p.Extra = string(extraJson)

		return p, nil
	}

	logger.WithFields(logger.Fields{
		"error": err,
	}).Warnf("log application failed, falling back to RPC")

	extensions := map[common.Address]ekubo_pool.Extension{
		poolKey.Config.Extension: staticExtra.Extension,
	}

	pools, err := fetchPools(ctx, d.ethrpcClient, d.config.DataFetcher, []quoting.PoolKey{poolKey}, extensions, nil)
	if err != nil {
		return p, fmt.Errorf("fetching pool state: %w", err)
	}

	if len(pools) == 0 {
		return entity.Pool{}, errors.New("failed to fetch pool from RPC")
	}

	return pools[0], nil
}

func (d *PoolTracker) applyLogs(logs []types.Log, poolKey quoting.PoolKey, poolState *quoting.PoolState) error {
	for _, log := range logs {
		if d.coreAddress.Cmp(log.Address) != 0 {
			continue
		}

		if log.Removed {
			return errors.New("chain reorg")
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
	}

	return nil
}

func handleSwappedEvent(data []byte, poolKey quoting.PoolKey, poolState *quoting.PoolState) error {
	n := new(big.Int).SetBytes(data)

	poolId := new(big.Int).And(
		new(big.Int).Rsh(n, 512),
		math.U256Max,
	)

	expectedPoolId, err := poolKey.NumId()
	if err != nil {
		return fmt.Errorf("computing expected pool id: %w", err)
	}

	if expectedPoolId.Cmp(poolId) != 0 {
		return nil
	}

	tickRaw := new(big.Int).And(
		n,
		math.U32Max,
	)

	if tickRaw.Cmp(math.TwoPow31) == -1 {
		poolState.ActiveTick = int32(tickRaw.Uint64())
	} else {
		poolState.ActiveTick = int32(tickRaw.Sub(tickRaw, math.TwoPow32).Int64())
	}
	n.Rsh(n, 32)

	sqrtRatioAfterCompact := new(big.Int).And(
		n,
		math.U96Max,
	)
	n.Rsh(n, 96)

	poolState.SqrtRatio = math.FloatSqrtRatioToFixed(sqrtRatioAfterCompact)

	poolState.Liquidity.And(
		n,
		math.U128Max,
	)

	return nil
}

func handlePositionUpdatedEvent(data []byte, poolKey quoting.PoolKey, poolState *quoting.PoolState) error {
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
