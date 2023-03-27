package synthetix

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core"
	poolPkg "github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/core/pool"
)

func Validate(route core.Route) error {
	var (
		poolStateVersion        PoolStateVersion
		blockTimestamp          uint64
		atomicMaxVolumePerBlock *big.Int
		lastAtomicVolume        *ExchangeVolumeAtPeriod
	)

	// Get PoolStateVersion, BlockTimestamp, AtomicMaxVolumePerBlock and LastAtomicVolume
	for _, path := range route.Paths {
		for _, p := range path.Pools {
			if p.GetType() == constant.PoolTypes.Synthetix {
				synthetixPool, ok := p.(*Pool)
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

	poolByAddress := make(map[string]poolPkg.IPool, len(route.OriginalPools))
	for _, originalPool := range route.OriginalPools {
		poolByAddress[originalPool.GetAddress()] = originalPool
	}

	for _, path := range route.Paths {
		var tokenAmountIn = path.Input
		for i := range path.Pools {
			pool, ok := poolByAddress[path.Pools[i].GetAddress()]
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
			}
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
