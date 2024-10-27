package curve

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/bytedance/sonic"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func (d *PoolsListUpdater) getNewPoolsTypeMeta(
	ctx context.Context,
	poolAndRegistries []PoolAndRegistries,
) ([]entity.Pool, error) {
	var (
		basePools       = make([]common.Address, len(poolAndRegistries))
		coins           = make([][8]common.Address, len(poolAndRegistries))
		underlyingCoins = make([][8]common.Address, len(poolAndRegistries))
		decimals        = make([][8]*big.Int, len(poolAndRegistries))
		aList           = make([]*big.Int, len(poolAndRegistries))
		aPreciseList    = make([]*big.Int, len(poolAndRegistries))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	for i, poolAndRegistry := range poolAndRegistries {
		if strings.EqualFold(poolAndRegistry.RegistryOrFactoryAddress, d.config.MetaPoolsFactoryAddress) {
			calls.AddCall(&ethrpc.Call{
				ABI:    metaPoolFactoryABI,
				Target: d.config.MetaPoolsFactoryAddress,
				Method: registryOrFactoryMethodGetBasePool,
				Params: []interface{}{poolAndRegistry.PoolAddress},
			}, []interface{}{&basePools[i]})
		} else {
			calls.AddCall(&ethrpc.Call{
				ABI:    metaABI,
				Target: poolAndRegistry.PoolAddress.Hex(),
				Method: poolMethodBasePool,
				Params: nil,
			}, []interface{}{&basePools[i]})
		}

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetCoins,
			Params: []interface{}{poolAndRegistry.PoolAddress},
		}, []interface{}{&coins[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetUnderlyingCoins,
			Params: []interface{}{poolAndRegistry.PoolAddress},
		}, []interface{}{&underlyingCoins[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetDecimals,
			Params: []interface{}{poolAndRegistry.PoolAddress},
		}, []interface{}{&decimals[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    metaABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodA,
			Params: nil,
		}, []interface{}{&aList[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    metaABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodAPrecise,
			Params: nil,
		}, []interface{}{&aPreciseList[i]})
	}
	if _, err := calls.TryAggregate(); err != nil {
		logger.Errorf("failed to aggregate call to get pool data, err: %v", err)
		return nil, err
	}

	aPrecisions, err := getAPrecisions(aList, aPreciseList)
	if err != nil {
		logger.Errorf("failed to calculate aPrecisions, err: %v", err)
		return nil, err
	}

	var pools = make([]entity.Pool, len(poolAndRegistries))
	for i := range poolAndRegistries {
		var reserves entity.PoolReserves
		var tokens []*entity.PoolToken
		var staticExtra = PoolMetaStaticExtra{
			LpToken:          strings.ToLower(poolAndRegistries[i].PoolAddress.Hex()),
			BasePool:         strings.ToLower(basePools[i].Hex()),
			RateMultiplier:   new(big.Int).Exp(big.NewInt(10), new(big.Int).Sub(big.NewInt(36), decimals[i][0]), nil).String(),
			APrecision:       aPrecisions[i].String(),
			UnderlyingTokens: extractNonZeroAddressesToStrings(underlyingCoins[i]),
		}
		for j := range coins[i] {
			coinAddress := convertToEtherAddress(coins[i][j].Hex(), d.config.ChainID)
			if strings.EqualFold(coinAddress, addressZero) {
				break
			}
			precision := new(big.Int).Exp(big.NewInt(10), new(big.Int).Sub(big.NewInt(18), decimals[i][j]), nil)
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, precision.String())
			staticExtra.Rates = append(staticExtra.Rates, "")
			reserves = append(reserves, zeroString)
			tokens = append(tokens, &entity.PoolToken{
				Address:   strings.ToLower(coinAddress),
				Weight:    defaultWeight,
				Swappable: true,
			})
		}
		staticExtraBytes, err := sonic.Marshal(staticExtra)
		if err != nil {
			logger.Errorf("failed to marshal static extra data, err: %v", err)
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:     strings.ToLower(poolAndRegistries[i].PoolAddress.Hex()),
			Exchange:    DexTypeCurve,
			Type:        PoolTypeMeta,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}
	}

	return pools, nil
}

func (d *PoolTracker) getNewPoolStateTypeMeta(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("[Curve] Start getting new state of pool %v with type %v", p.Address, p.Type)

	var (
		initialA, futureA, initialATime, futureATime, swapFee, adminFee, lpSupply *big.Int
		balances                                                                  = make([]*big.Int, len(p.Tokens))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    metaABI,
		Target: p.Address,
		Method: poolMethodInitialA,
		Params: nil,
	}, []interface{}{&initialA})

	calls.AddCall(&ethrpc.Call{
		ABI:    metaABI,
		Target: p.Address,
		Method: poolMethodFutureA,
		Params: nil,
	}, []interface{}{&futureA})

	calls.AddCall(&ethrpc.Call{
		ABI:    metaABI,
		Target: p.Address,
		Method: poolMethodInitialATime,
		Params: nil,
	}, []interface{}{&initialATime})

	calls.AddCall(&ethrpc.Call{
		ABI:    metaABI,
		Target: p.Address,
		Method: poolMethodFutureATime,
		Params: nil,
	}, []interface{}{&futureATime})

	calls.AddCall(&ethrpc.Call{
		ABI:    metaABI,
		Target: p.Address,
		Method: poolMethodFee,
		Params: nil,
	}, []interface{}{&swapFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    metaABI,
		Target: p.Address,
		Method: poolMethodAdminFee,
		Params: nil,
	}, []interface{}{&adminFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: p.GetLpToken(),
		Method: erc20MethodTotalSupply,
		Params: nil,
	}, []interface{}{&lpSupply})

	for i := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    metaABI,
			Target: p.Address,
			Method: poolMethodBalances,
			Params: []interface{}{big.NewInt(int64(i))},
		}, []interface{}{&balances[i]})
	}

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
			"error":       err,
		}).Errorf("failed to aggregate call pool data")
		return entity.Pool{}, err
	}

	var extra = PoolMetaExtra{
		InitialA:     initialA.String(),
		FutureA:      futureA.String(),
		InitialATime: initialATime.Int64(),
		FutureATime:  futureATime.Int64(),
		SwapFee:      swapFee.String(),
		AdminFee:     adminFee.String(),
	}

	extraBytes, err := sonic.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
			"error":       err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	var reserves = make(entity.PoolReserves, 0, len(balances)+1)
	for i := range balances {
		reserves = append(reserves, safeCastBigIntToReserve(balances[i]))
	}
	reserves = append(reserves, safeCastBigIntToReserve(lpSupply))

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = reserves

	logger.Infof("[Curve] Finish getting new state of pool %v with type %v", p.Address, p.Type)

	return p, nil
}
