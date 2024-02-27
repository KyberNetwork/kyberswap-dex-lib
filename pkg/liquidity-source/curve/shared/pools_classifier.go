package shared

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
)

func (u *PoolsListUpdater) ClassifyPools(ctx context.Context, dataSource CurveDataSource, pools []CurvePool) (map[string]CurvePoolType, error) {
	if dataSource == CURVE_DATASOURCE_MAIN {
		return u.classifyPoolsFromMainRegistry(ctx, dataSource, pools)
	}
	if dataSource == CURVE_DATASOURCE_CRYPTO {
		return u.classifyPoolsFromCryptoRegistry(ctx, dataSource, pools)
	}
	return u.classifyPoolsFromFactory(ctx, dataSource, pools)
}

func (u *PoolsListUpdater) classifyPoolsFromCryptoRegistry(ctx context.Context, dataSource CurveDataSource, pools []CurvePool) (map[string]CurvePoolType, error) {
	typeMap := make(map[string]CurvePoolType, len(pools))
	for _, pool := range pools {
		if pool.IsMetaPool {
			typeMap[pool.Address] = CURVE_POOL_TYPE_CRYPTO_META
		} else {
			typeMap[pool.Address] = CURVE_POOL_TYPE_CRYPTO
		}
	}
	return typeMap, nil
}

// (only for factory-xxx source, not `main` and `crypto`)
// https://github.com/curvefi/curve-js/blob/cb26335/src/factory/factory-api.ts#L72
func (u *PoolsListUpdater) classifyPoolsFromFactory(ctx context.Context, dataSource CurveDataSource, pools []CurvePool) (map[string]CurvePoolType, error) {
	typeMap := make(map[string]CurvePoolType, len(pools))

	isCrypto := dataSource == CURVE_DATASOURCE_FACTORY_CRYPTO || dataSource == CURVE_DATASOURCE_FACTORY_TRICRYPTO

	for _, pool := range pools {
		if isCrypto {
			if pool.IsMetaPool {
				// only factory-crypto has meta pool, factory-tricrypto doesn't (yet), anw we have the flag already
				typeMap[pool.Address] = CURVE_POOL_TYPE_CRYPTO_META
			} else {
				typeMap[pool.Address] = CURVE_POOL_TYPE_CRYPTO
			}
		} else if pool.IsMetaPool {
			if dataSource == CURVE_DATASOURCE_FACTORY_STABLE_NG {
				typeMap[pool.Address] = CURVE_POOL_TYPE_STABLE_NG_META
			} else {
				typeMap[pool.Address] = CURVE_POOL_TYPE_STABLE_META
			}
		} else {
			if dataSource == CURVE_DATASOURCE_FACTORY_STABLE_NG {
				typeMap[pool.Address] = CURVE_POOL_TYPE_STABLE_NG_PLAIN
			} else {
				typeMap[pool.Address] = CURVE_POOL_TYPE_STABLE_PLAIN
			}
		}

		u.logger.Debugf("classifyPoolsFromFactory %v: %v %v %v", pool.Address, typeMap[pool.Address], isCrypto, pool.IsMetaPool)
	}

	return typeMap, nil
}

func (u *PoolsListUpdater) classifyPoolsFromMainRegistry(ctx context.Context, dataSource CurveDataSource, pools []CurvePool) (map[string]CurvePoolType, error) {
	typeMap := make(map[string]CurvePoolType, len(pools))

	// Curve's offchain API doesn't have enough info, so fetch them via RPC
	gammaList := make([]*big.Int, len(pools))
	underlyingCoins128 := make([][MaxTokenCount]common.Address, len(pools))
	underlyingCoins256 := make([][MaxTokenCount]common.Address, len(pools))

	calls := u.ethrpcClient.NewRequest().SetContext(ctx).SetRequireSuccess(false)

	for poolIdx, pool := range pools {
		for coinIdx := range pool.Coins {
			calls.AddCall(&ethrpc.Call{
				ABI:    underlyingCoins128ABI,
				Target: pool.Address,
				Method: poolMethodUnderlyingCoins,
				Params: []interface{}{big.NewInt(int64(coinIdx))},
			}, []interface{}{&underlyingCoins128[poolIdx][coinIdx]})

			calls.AddCall(&ethrpc.Call{
				ABI:    underlyingCoins256ABI,
				Target: pool.Address,
				Method: poolMethodUnderlyingCoins,
				Params: []interface{}{big.NewInt(int64(coinIdx))},
			}, []interface{}{&underlyingCoins256[poolIdx][coinIdx]})
		}

		calls.AddCall(&ethrpc.Call{
			ABI:    gammaABI,
			Target: pool.Address,
			Method: poolMethodGamma,
			Params: nil,
		}, []interface{}{&gammaList[poolIdx]})
	}

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate to get pool data")
		return nil, err
	}

	for poolIdx, pool := range pools {
		if gammaList[poolIdx] != nil {
			if pool.IsMetaPool {
				typeMap[pool.Address] = CURVE_POOL_TYPE_CRYPTO_META
			} else {
				typeMap[pool.Address] = CURVE_POOL_TYPE_CRYPTO
			}
			continue
		}

		if pool.IsMetaPool {
			typeMap[pool.Address] = CURVE_POOL_TYPE_STABLE_META
			continue
		}

		// pure plain pools don't have underlying coins
		hasUnderlyingCoins := true
		for coinIdx := range pool.Coins {
			if strings.EqualFold(underlyingCoins128[poolIdx][coinIdx].Hex(), valueobject.ZeroAddress) &&
				strings.EqualFold(underlyingCoins256[poolIdx][coinIdx].Hex(), valueobject.ZeroAddress) {
				hasUnderlyingCoins = false
			}
		}
		if !hasUnderlyingCoins {
			typeMap[pool.Address] = CURVE_POOL_TYPE_STABLE_PLAIN
			continue
		}

		// we might still re-use curve-stable-plain for Lending pool (Compound/Aave/Yearn/Cream)
		// by fetching stored_rates upfront (and maybe supporting dynamic fee)
		// but for now just set to stable-lending and don't care about them
		typeMap[pool.Address] = CURVE_POOL_TYPE_STABLE_LENDING
	}
	return typeMap, nil
}
