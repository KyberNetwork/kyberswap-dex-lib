package synthetix

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// Validate will do all the swap based on Route's paths.
// It will update the pools in the process of doing so hence will only take a copy
func Validate(poolByAddress map[string]poolPkg.IPool, route *valueobject.Route) error {
	var (
		poolStateVersion        PoolStateVersion
		blockTimestamp          uint64
		atomicMaxVolumePerBlock *big.Int
		lastAtomicVolume        *ExchangeVolumeAtPeriod
	)
	poolBucket := valueobject.NewPoolBucket(poolByAddress)

	// Get PoolStateVersion, BlockTimestamp, AtomicMaxVolumePerBlock and LastAtomicVolume
	for _, path := range route.Paths {
		for _, poolAddress := range path.PoolAddresses {
			pool, ok := poolBucket.GetPool(poolAddress)
			if !ok {
				continue
			}
			if pool.GetType() == constant.PoolTypes.Synthetix {
				synthetixPool, ok := pool.(*Pool)
				if !ok {
					continue
				}

				poolStateVersion = synthetixPool.poolStateVersion
				blockTimestamp = synthetixPool.poolState.BlockTimestamp
				atomicMaxVolumePerBlock = synthetixPool.poolState.AtomicMaxVolumePerBlock
				lastAtomicVolume = synthetixPool.poolState.LastAtomicVolume

				break
			}
		}
	}

	// Normal Synthetix pool does not have to validate the total volume
	if poolStateVersion != PoolStateVersionAtomic {
		return nil
	}

	totalVolume := constant.Zero

	for _, path := range route.Paths {
		var tokenAmountIn = path.Input
		for i, poolAddress := range path.PoolAddresses {
			pool, ok := poolBucket.GetPool(poolAddress)
			if !ok {
				continue
			}

			calcAmountOutResult, err := pool.CalcAmountOut(
				tokenAmountIn,
				path.Tokens[i+1].Address,
			)
			if err != nil {
				return err
			}
			tokenAmountOut := calcAmountOutResult.TokenAmountOut
			if tokenAmountOut == nil || tokenAmountOut.Amount == nil || tokenAmountOut.Amount.Cmp(constant.Zero) <= 0 {
				return err
			}

			synthetixPool, ok := pool.(*Pool)
			if ok {
				synthetixTradeVolume, err := synthetixPool.GetAtomicVolume(tokenAmountIn, path.Tokens[i+1].Address)
				if err != nil {
					return err
				}

				if synthetixTradeVolume != nil && synthetixTradeVolume.Cmp(constant.Zero) > 0 {
					totalVolume = new(big.Int).Add(totalVolume, synthetixTradeVolume)
				}
			}
			updateBalanceParams := poolPkg.UpdateBalanceParams{
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
	lastAtomicVolume *ExchangeVolumeAtPeriod,
) error {
	if lastAtomicVolume == nil {
		return ErrInvalidLastAtomicVolume
	}

	currentVolume := sourceSusdValue

	if lastAtomicVolume.Time == blockTimestamp {
		currentVolume = new(big.Int).Add(lastAtomicVolume.Volume, sourceSusdValue)
	}

	if currentVolume.Cmp(atomicMaxVolumePerBlock) > 0 {
		return ErrSurpassedVolumeLimit
	}

	return nil
}
