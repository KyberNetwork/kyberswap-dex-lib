package curve

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func (d *PoolsListUpdater) getNewPoolsTypePlainOracle(
	ctx context.Context,
	poolAndRegistries []PoolAndRegistries,
) ([]entity.Pool, error) {
	var (
		coins        = make([][8]common.Address, len(poolAndRegistries))
		decimals     = make([][8]*big.Int, len(poolAndRegistries))
		aList        = make([]*big.Int, len(poolAndRegistries))
		aPreciseList = make([]*big.Int, len(poolAndRegistries))
		plainOracles = make([]common.Address, len(poolAndRegistries))
		lpAddresses  = make([]common.Address, len(poolAndRegistries))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	for i, poolAndRegistry := range poolAndRegistries {
		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetCoins,
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&coins[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetDecimals,
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&decimals[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    plainOracleABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodA,
		}, []any{&aList[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    plainOracleABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodAPrecise,
		}, []any{&aPreciseList[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    plainOracleABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: plainOracleMethodOracle,
		}, []any{&plainOracles[i]})

		calls.AddCall(&ethrpc.Call{
			ABI:    mainRegistryABI,
			Target: d.config.MainRegistryAddress,
			Method: registryOrFactoryMethodGetLpToken,
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&lpAddresses[i]})
	}
	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to aggregate call to get pool data")
		return nil, err
	}

	aPrecisions, err := getAPrecisions(aList, aPreciseList)
	if err != nil {
		logger.WithFields(logger.Fields{
			"error": err,
		}).Errorf("failed to calculate aPrecisions")
		return nil, err
	}

	var pools = make([]entity.Pool, len(poolAndRegistries))
	for i := range poolAndRegistries {
		var reserves entity.PoolReserves
		var tokens []*entity.PoolToken
		var staticExtra = PoolPlainOracleStaticExtra{
			LpToken:    strings.ToLower(lpAddresses[i].Hex()),
			APrecision: aPrecisions[i].String(),
			Oracle:     plainOracles[i].Hex(),
		}
		// The curve-base found inside the metaFactory has the lpToken equals its own pool Address and has the totalSupply method.
		if strings.EqualFold(staticExtra.LpToken, addressZero) && strings.EqualFold(poolAndRegistries[i].RegistryOrFactoryAddress, d.config.MetaPoolsFactoryAddress) {
			staticExtra.LpToken = strings.ToLower(poolAndRegistries[i].PoolAddress.Hex())
		}
		for j := range coins[i] {
			coinAddress := convertToEtherAddress(coins[i][j].Hex(), d.config.ChainID)
			if strings.EqualFold(coinAddress, addressZero) {
				break
			}
			precision := new(big.Int).Exp(big.NewInt(10), new(big.Int).Sub(big.NewInt(18), decimals[i][j]), nil)
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, precision.String())
			reserves = append(reserves, zeroString)
			tokens = append(tokens, &entity.PoolToken{
				Address:   strings.ToLower(coinAddress),
				Swappable: true,
			})
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to marshal static extra data")
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:     strings.ToLower(poolAndRegistries[i].PoolAddress.Hex()),
			Exchange:    DexTypeCurve,
			Type:        PoolTypePlainOracle,
			Timestamp:   time.Now().Unix(),
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}
	}

	return pools, nil
}

func (d *PoolTracker) getNewPoolStateTypePlainOracle(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.Infof("[Curve] Start getting new state of pool %v with type %v", p.Address, p.Type)

	var (
		plainOraclePoolPrecision                                                                      = bignumber.TenPowInt(18)
		staticExtra                                                                                   PoolPlainOracleStaticExtra
		initialA, futureA, initialATime, futureATime, swapFee, adminFee, oracleLatestAnswer, lpSupply *big.Int
		balances                                                                                      = make([]*big.Int, len(p.Tokens))
	)

	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to unmarshal static extra data")
		return entity.Pool{}, err
	}

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		calls.SetOverrides(overrides)
	}

	calls.AddCall(&ethrpc.Call{
		ABI:    plainOracleABI,
		Target: p.Address,
		Method: poolMethodInitialA,
	}, []any{&initialA})

	calls.AddCall(&ethrpc.Call{
		ABI:    plainOracleABI,
		Target: p.Address,
		Method: poolMethodFutureA,
	}, []any{&futureA})

	calls.AddCall(&ethrpc.Call{
		ABI:    plainOracleABI,
		Target: p.Address,
		Method: poolMethodInitialATime,
	}, []any{&initialATime})

	calls.AddCall(&ethrpc.Call{
		ABI:    plainOracleABI,
		Target: p.Address,
		Method: poolMethodFutureATime,
	}, []any{&futureATime})

	calls.AddCall(&ethrpc.Call{
		ABI:    plainOracleABI,
		Target: p.Address,
		Method: poolMethodFee,
	}, []any{&swapFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    plainOracleABI,
		Target: p.Address,
		Method: poolMethodAdminFee,
		Params: nil,
	}, []any{&adminFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    plainOracleABI,
		Target: p.Address,
		Method: poolMethodAdminFee,
	}, []any{&adminFee})

	calls.AddCall(&ethrpc.Call{
		ABI:    oracleABI,
		Target: staticExtra.Oracle,
		Method: oracleMethodLatestAnswer,
	}, []any{&oracleLatestAnswer})

	calls.AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: p.GetLpToken(),
		Method: erc20MethodTotalSupply,
	}, []any{&lpSupply})

	for i := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    baseABI,
			Target: p.Address,
			Method: poolMethodBalances,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&balances[i]})
	}

	if _, err := calls.Aggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
			"error":       err,
		}).Errorf("failed to aggregate call pool data")
		return entity.Pool{}, err
	}

	var extra = PoolPlainOracleExtra{
		InitialA:     safeCastBigIntToString(initialA),
		FutureA:      safeCastBigIntToString(futureA),
		InitialATime: safeCastBigIntToInt64(initialATime),
		FutureATime:  safeCastBigIntToInt64(futureATime),
		SwapFee:      safeCastBigIntToString(swapFee),
		AdminFee:     safeCastBigIntToString(adminFee),
		Rates: []*big.Int{
			plainOraclePoolPrecision,
			oracleLatestAnswer,
		},
	}
	extraBytes, err := json.Marshal(extra)
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
