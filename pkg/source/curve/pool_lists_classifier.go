package curve

import (
	"context"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func (d *PoolsListUpdater) classifyPoolTypes(
	ctx context.Context,
	poolsSourceIndex int,
	registryOrFactoryABI abi.ABI,
	registryOrFactoryAddress string,
	poolAddresses []common.Address,
) ([]string, error) {
	switch poolsSourceIndex {
	case sourceMainRegistry:
		return d.classifyPoolsFromMainRegistry(ctx, registryOrFactoryABI, registryOrFactoryAddress, poolAddresses)
	case sourceMetaPoolsFactory:
		return d.classifyPoolsFromMetaPoolsFactory(ctx, registryOrFactoryABI, registryOrFactoryAddress, poolAddresses)
	case sourceCryptoPoolsRegistry:
		return d.classifyCurveV2PoolTypes(ctx, registryOrFactoryABI, registryOrFactoryAddress, poolAddresses)
	case sourceCryptoPoolsFactory:
		return d.classifyCurveV2PoolTypes(ctx, registryOrFactoryABI, registryOrFactoryAddress, poolAddresses)
	case sourceMetaRegistry:
		return d.classifyPoolsFromMainRegistry(ctx, registryOrFactoryABI, registryOrFactoryAddress, poolAddresses)
	default:
		// Index can be found here https://github.com/KyberNetwork/kyberswap-dex-lib/blob/0e4796ffde08481ef8b456e354cf2cb7b3aa8268/pkg/source/curve/pools_list_updater.go#L69-L79
		logger.Errorf("unknown pools source index %v", poolsSourceIndex)
		return nil, nil
	}
}

// classifyPoolsFromMainRegistry includes plainOracle, base, meta, aave, compound
func (d *PoolsListUpdater) classifyPoolsFromMainRegistry(
	ctx context.Context,
	registryOrFactoryABI abi.ABI,
	registryOrFactoryAddress string,
	poolAddresses []common.Address,
) ([]string, error) {
	var coins = make([][8]common.Address, len(poolAddresses))
	var underlyingCoins = make([][8]common.Address, len(poolAddresses))
	var aaveSignatures = make([]*big.Int, len(poolAddresses))
	var plainOracleSignatures = make([]common.Address, len(poolAddresses))
	var isMetaList = make([]bool, len(poolAddresses))
	var gammaList = make([]*big.Int, len(poolAddresses))

	calls := d.ethrpcClient.NewRequest().SetContext(ctx).SetRequireSuccess(false)

	for i, poolAddress := range poolAddresses {
		calls.AddCall(&ethrpc.Call{
			ABI:    registryOrFactoryABI,
			Target: registryOrFactoryAddress,
			Method: registryOrFactoryMethodGetCoins,
			Params: []interface{}{poolAddress},
		}, []interface{}{&coins[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    registryOrFactoryABI,
			Target: registryOrFactoryAddress,
			Method: registryOrFactoryMethodGetUnderlyingCoins,
			Params: []interface{}{poolAddress},
		}, []interface{}{&underlyingCoins[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    registryOrFactoryABI,
			Target: registryOrFactoryAddress,
			Method: registryOrFactoryMethodIsMeta,
			Params: []interface{}{poolAddress},
		}, []interface{}{&isMetaList[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    aaveABI,
			Target: poolAddresses[i].Hex(),
			Method: aaveMethodOffpegFeeMultiplier,
			Params: nil,
		}, []interface{}{&aaveSignatures[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    plainOracleABI,
			Target: poolAddresses[i].Hex(),
			Method: plainOracleMethodOracle,
			Params: nil,
		}, []interface{}{&plainOracleSignatures[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    twoABI,
			Target: poolAddresses[i].Hex(),
			Method: poolMethodGamma,
			Params: nil,
		}, []interface{}{&gammaList[i]})
	}

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate to get pool data")
		return nil, err
	}

	var poolTypes = make([]string, len(poolAddresses))
	for i := range poolAddresses {
		if gammaList[i] != nil {
			if d.isTwo(coins[i]) {
				poolTypes[i] = poolTypeTwo
			} else {
				poolTypes[i] = poolTypeTricrypto
			}
			continue
		}

		if isMetaList[i] {
			poolTypes[i] = poolTypeMeta
			continue
		}

		if d.isPlainOraclePool(plainOracleSignatures[i]) {
			poolTypes[i] = poolTypePlainOracle
			continue
		}

		if d.isAavePool(aaveSignatures[i], underlyingCoins[i]) {
			poolTypes[i] = poolTypeAave
			continue
		}

		ok, err := d.isCompoundPool(ctx, poolAddresses[i], coins[i], underlyingCoins[i])
		if err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": poolAddresses[i],
				"error":       err,
			}).Errorf("failed to detect compound pool type")

			return nil, err
		}
		if ok {
			poolTypes[i] = poolTypeCompound
			continue
		}

		if d.isBasePool(coins[i], underlyingCoins[i]) {
			poolTypes[i] = poolTypeBase
			continue
		}

		poolTypes[i] = poolTypeLending
	}

	return poolTypes, nil
}

// classifyPoolsFromMetaPoolsFactory includes meta only
func (d *PoolsListUpdater) classifyPoolsFromMetaPoolsFactory(
	ctx context.Context,
	registryOrFactoryABI abi.ABI,
	registryOrFactoryAddress string,
	poolAddresses []common.Address,
) ([]string, error) {
	var isMetaList = make([]bool, len(poolAddresses))

	calls := d.ethrpcClient.NewRequest().SetContext(ctx).SetRequireSuccess(false)

	for i, poolAddress := range poolAddresses {
		calls.AddCall(&ethrpc.Call{
			ABI:    registryOrFactoryABI,
			Target: registryOrFactoryAddress,
			Method: registryOrFactoryMethodIsMeta,
			Params: []interface{}{poolAddress},
		}, []interface{}{&isMetaList[i]})
	}

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate to get pool data")
		return nil, err
	}

	var poolTypes = make([]string, len(poolAddresses))
	for i := range poolAddresses {
		if isMetaList[i] {
			poolTypes[i] = poolTypeMeta
		}
	}

	return poolTypes, nil
}

// classifyCurveV2PoolTypes includes two and tricrypto
func (d *PoolsListUpdater) classifyCurveV2PoolTypes(
	ctx context.Context,
	registryOrFactoryABI abi.ABI,
	registryOrFactoryAddress string,
	poolAddresses []common.Address,
) ([]string, error) {
	coins := make([][8]common.Address, len(poolAddresses))
	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i, poolAddress := range poolAddresses {
		calls.AddCall(&ethrpc.Call{
			ABI:    registryOrFactoryABI,
			Target: registryOrFactoryAddress,
			Method: registryOrFactoryMethodGetCoins,
			Params: []interface{}{poolAddress},
		}, []interface{}{&coins[i]})
	}
	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to get coins data of pool")
		return nil, err
	}

	var poolTypes = make([]string, len(poolAddresses))
	for i := range poolAddresses {
		if d.isTwo(coins[i]) {
			poolTypes[i] = poolTypeTwo
		} else if d.isTricrypto(coins[i]) {
			poolTypes[i] = poolTypeTricrypto
		} else {
			logger.Infof("unsupported curve v2 pool: %s", poolAddresses[i].Hex())
			poolTypes[i] = poolTypeUnsupported
		}
	}

	return poolTypes, nil
}

// isBasePool PlainOraclePool should
// be a BasePool but having method "oracle" in its contract
func (d *PoolsListUpdater) isPlainOraclePool(oracleAddress common.Address) bool {
	return !strings.EqualFold(oracleAddress.String(), addressZero)
}

// isBasePool BasePool should
// have underlying coins equals coins
func (d *PoolsListUpdater) isBasePool(coins [8]common.Address, underlyingCoins [8]common.Address) bool {
	for i := 0; i < len(coins); i++ {
		if !strings.EqualFold(underlyingCoins[i].Hex(), addressZero) && !strings.EqualFold(coins[i].Hex(), underlyingCoins[i].Hex()) {
			return false
		}
	}
	return true
}

// isAavePool AavePool should
// have underlying coins and not native coin
// have method "offpeg_fee_multiplier" in its contract
func (d *PoolsListUpdater) isAavePool(aaveSignature *big.Int, underlyingCoins [8]common.Address) bool {
	if strings.EqualFold(underlyingCoins[0].Hex(), addressZero) {
		return false
	}
	for i := range underlyingCoins {
		if strings.EqualFold(underlyingCoins[i].Hex(), addressEther) {
			return false
		}
	}
	return aaveSignature != nil
}

// isCompoundPool CompoundPool should
// have underlying coins and not native coin
// have at least 1 coin is compoundToken (has "compound" in token name)
func (d *PoolsListUpdater) isCompoundPool(
	ctx context.Context,
	poolAddress common.Address,
	coins [8]common.Address,
	underlyingCoins [8]common.Address,
) (bool, error) {
	if strings.EqualFold(underlyingCoins[0].Hex(), addressZero) {
		return false, nil
	}
	for i := range underlyingCoins {
		if strings.EqualFold(underlyingCoins[i].Hex(), addressEther) {
			return false, nil
		}
	}
	var tokenNames = make([]string, len(coins))
	var isCToken = make([]bool, len(coins))
	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	for i, coin := range coins {
		if strings.EqualFold(coin.Hex(), addressZero) {
			break
		}
		if strings.EqualFold(coin.Hex(), addressEther) {
			continue
		}
		calls.AddCall(&ethrpc.Call{
			ABI:    cerc20ABI,
			Target: coin.Hex(),
			Method: cerc20MethodName,
			Params: nil,
		}, []interface{}{&tokenNames[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    cerc20ABI,
			Target: coin.Hex(),
			Method: cerc20MethodIsCToken,
			Params: nil,
		}, []interface{}{&isCToken[i]})
	}

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": poolAddress,
			"error":       err,
		}).Errorf("failed to get coin name")
		return false, err
	}

	for i, name := range tokenNames {
		if strings.Contains(strings.ToLower(name), "compound") && isCToken[i] {
			return true, nil
		}
	}

	return false, nil
}

// isTwo TwoCryptoPool
// is curveV2, belongs to CryptoFactory and CryptoRegistry
// has "gamma" in its contracts
// has only 2 coins (has 3 coins is TricryptoPool)
func (d *PoolsListUpdater) isTwo(coins [8]common.Address) bool {
	var numberOfCoin = 0
	for _, coin := range coins {
		if strings.EqualFold(coin.Hex(), addressZero) {
			break
		}
		numberOfCoin += 1
	}

	return numberOfCoin == 2
}

// isTricrypto is curveV2, belongs to CryptoRegistry and CryptoFactory
// has "gamma" in its contracts
// has only 3 coins (has 2 coins is TwoPool)
func (d *PoolsListUpdater) isTricrypto(coins [8]common.Address) bool {
	var numberOfCoin = 0
	for _, coin := range coins {
		if strings.EqualFold(coin.Hex(), addressZero) {
			break
		}
		numberOfCoin += 1
	}

	return numberOfCoin == 3
}
