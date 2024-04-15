package synthetix

import (
	"context"
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/pooltypes"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/synthetix"
	"github.com/KyberNetwork/logger"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/metrics"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type SynthetixValidator struct {
}

func NewSynthetixValidator() *SynthetixValidator {
	return &SynthetixValidator{}
}

// Validate will reapply pool update and will have to modify the pool state. Do not use original pools for this
func (v *SynthetixValidator) Validate(ctx context.Context, poolByAddress map[string]poolpkg.IPoolSimulator, route *valueobject.Route) error {
	err := Validate(poolByAddress, route)

	if errors.Is(err, synthetix.ErrInvalidLastAtomicVolume) {
		return err
	}

	if errors.Is(err, synthetix.ErrSurpassedVolumeLimit) {
		logger.Error("invalid Synthetix volume for route")

		metrics.IncrInvalidSynthetixVolume(ctx)
	}

	return nil
}

// Validate will do all the swap based on Route's paths.
// It will update the pools in the process of doing so hence will only take a copy
func Validate(poolByAddress map[string]poolpkg.IPoolSimulator, route *valueobject.Route) error {
	var (
		poolStateVersion        synthetix.PoolStateVersion
		blockTimestamp          uint64
		atomicMaxVolumePerBlock *big.Int
		lastAtomicVolume        *synthetix.ExchangeVolumeAtPeriod
	)
	poolBucket := valueobject.NewPoolBucket(poolByAddress)

	// Get PoolStateVersion, BlockTimestamp, AtomicMaxVolumePerBlock and LastAtomicVolume
	for _, path := range route.Paths {
		for _, poolAddress := range path.PoolAddresses {
			pool, ok := poolBucket.GetPool(poolAddress)
			if !ok {
				continue
			}
			if pool.GetType() == pooltypes.PoolTypes.Synthetix {
				synthetixPool, ok := pool.(*synthetix.PoolSimulator)
				if !ok {
					continue
				}

				poolStateVersion = synthetixPool.GetPoolStateVersion()
				blockTimestamp = synthetixPool.GetPoolState().BlockTimestamp
				atomicMaxVolumePerBlock = synthetixPool.GetPoolState().AtomicMaxVolumePerBlock
				lastAtomicVolume = synthetixPool.GetPoolState().LastAtomicVolume

				break
			}
		}
	}

	// Normal Synthetix pool does not have to validate the total volume
	if poolStateVersion != synthetix.PoolStateVersionAtomic {
		return nil
	}

	totalVolume := constant.Zero

	for _, path := range route.Paths {
		var tokenAmountIn = *path.Input.ToDexLibAmount()
		for i, poolAddress := range path.PoolAddresses {
			pool, ok := poolBucket.GetPool(poolAddress)
			if !ok {
				continue
			}

			calcAmountOutResult, err := poolpkg.CalcAmountOut(
				pool,
				tokenAmountIn,
				path.Tokens[i+1].Address,
				nil,
			)

			if err != nil {
				return err
			}
			tokenAmountOut := calcAmountOutResult.TokenAmountOut
			if tokenAmountOut == nil || tokenAmountOut.Amount == nil || tokenAmountOut.Amount.Cmp(constant.Zero) <= 0 {
				return err
			}

			synthetixPool, ok := pool.(*synthetix.PoolSimulator)
			if ok {
				synthetixTradeVolume, err := synthetixPool.GetAtomicVolume(tokenAmountIn, path.Tokens[i+1].Address)
				if err != nil {
					return err
				}

				if synthetixTradeVolume != nil && synthetixTradeVolume.Cmp(constant.Zero) > 0 {
					totalVolume.Add(totalVolume, synthetixTradeVolume)
				}
			}

			//we need not inventories here since this is Synthetix's pools only
			updateBalanceParams := poolpkg.UpdateBalanceParams{
				TokenAmountIn:  tokenAmountIn,
				TokenAmountOut: *tokenAmountOut,
				Fee:            *calcAmountOutResult.Fee,
				SwapInfo:       calcAmountOutResult.SwapInfo,
			}
			// clone the pool before updating it, so it doesn't modify the original data copied from pool manager
			pool = poolBucket.ClonePool(poolAddress)

			// modify our copy
			pool.UpdateBalance(updateBalanceParams)

			tokenAmountIn = *tokenAmountOut
		}
	}

	return checkAtomicVolume(totalVolume, blockTimestamp, atomicMaxVolumePerBlock, lastAtomicVolume)
}

func checkAtomicVolume(
	sourceSusdValue *big.Int,
	blockTimestamp uint64,
	atomicMaxVolumePerBlock *big.Int,
	lastAtomicVolume *synthetix.ExchangeVolumeAtPeriod,
) error {
	if lastAtomicVolume == nil {
		return synthetix.ErrInvalidLastAtomicVolume
	}

	currentVolume := sourceSusdValue

	if lastAtomicVolume.Time == blockTimestamp {
		currentVolume = new(big.Int).Add(lastAtomicVolume.Volume, sourceSusdValue)
	}

	if currentVolume.Cmp(atomicMaxVolumePerBlock) > 0 {
		return synthetix.ErrSurpassedVolumeLimit
	}

	return nil
}
