package curve

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

func (d *PoolsListUpdater) getNewPoolsTypeMeta(
	ctx context.Context,
	poolAndRegistries []PoolAndRegistries,
) ([]entity.Pool, error) {
	var (
		basePools         = make([]common.Address, len(poolAndRegistries))
		poolBasePools     = make([]common.Address, len(poolAndRegistries))
		registryBasePools = make([]common.Address, len(poolAndRegistries))
		coins             = make([][8]common.Address, len(poolAndRegistries))
		underlyingCoins   = make([][8]common.Address, len(poolAndRegistries))
		decimals          = make([][8]*big.Int, len(poolAndRegistries))
		aList             = make([]*big.Int, len(poolAndRegistries))
		aPreciseList      = make([]*big.Int, len(poolAndRegistries))
	)

	calls := d.ethrpcClient.NewRequest().SetContext(ctx)

	for i, poolAndRegistry := range poolAndRegistries {
		// Base pool is resolved from up to two candidates; we pick the first non-zero
		// after aggregation (see below), so coverage is the union of all sources.
		// Candidate A: the pool's own base_pool() getter. Newer templates (e.g.
		// GUSD/3Crv) implement it; legacy metapools (MIM/FRAX/LUSD/TUSD-3Crv) do not,
		// so this call may revert and leave the slot zero (TryAggregate tolerates it).
		calls.AddCall(&ethrpc.Call{
			ABI:    metaABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodBasePool,
		}, []any{&poolBasePools[i]})

		// Candidate B: a registry that indexes base pools — the meta factory for
		// factory-listed pools, otherwise the MetaRegistry (which covers legacy pools
		// lacking base_pool()). Skipped when neither is configured (e.g. some chains).
		registryTarget := ""
		if strings.EqualFold(poolAndRegistry.RegistryOrFactoryAddress, d.config.MetaPoolsFactoryAddress) {
			registryTarget = d.config.MetaPoolsFactoryAddress
		} else if d.config.MetaRegistryAddress != "" && !strings.EqualFold(d.config.MetaRegistryAddress, addressZero) {
			registryTarget = d.config.MetaRegistryAddress
		}
		if registryTarget != "" {
			calls.AddCall(&ethrpc.Call{
				ABI:    metaPoolFactoryABI,
				Target: registryTarget,
				Method: registryOrFactoryMethodGetBasePool,
				Params: []any{poolAndRegistry.PoolAddress},
			}, []any{&registryBasePools[i]})
		}

		calls.AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetCoins,
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&coins[i]}).AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetUnderlyingCoins,
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&underlyingCoins[i]}).AddCall(&ethrpc.Call{
			ABI:    poolAndRegistry.RegistryOrFactoryABI,
			Target: poolAndRegistry.RegistryOrFactoryAddress,
			Method: registryOrFactoryMethodGetDecimals,
			Params: []any{poolAndRegistry.PoolAddress},
		}, []any{&decimals[i]}).AddCall(&ethrpc.Call{
			ABI:    metaABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodA,
		}, []any{&aList[i]}).AddCall(&ethrpc.Call{
			ABI:    metaABI,
			Target: poolAndRegistry.PoolAddress.Hex(),
			Method: poolMethodAPrecise,
		}, []any{&aPreciseList[i]})
	}
	if _, err := calls.TryAggregate(); err != nil {
		logger.Errorf("failed to aggregate call to get pool data, err: %v", err)
		return nil, err
	}

	// Prefer the registry answer, fall back to the pool's own base_pool() getter.
	// A base pool stays zero only if none of the sources know it.
	for i := range poolAndRegistries {
		basePools[i] = registryBasePools[i]
		if basePools[i] == (common.Address{}) {
			basePools[i] = poolBasePools[i]
		}
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
			LpToken:          hexutil.Encode(poolAndRegistries[i].PoolAddress[:]),
			BasePool:         hexutil.Encode(basePools[i][:]),
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
				Swappable: true,
			})
		}
		staticExtraBytes, err := json.Marshal(staticExtra)
		if err != nil {
			logger.Errorf("failed to marshal static extra data, err: %v", err)
			return nil, err
		}

		pools[i] = entity.Pool{
			Address:     hexutil.Encode(poolAndRegistries[i].PoolAddress[:]),
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

	calls := d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).SetFrom(AddrDummy).
		AddCall(&ethrpc.Call{
			ABI:    metaABI,
			Target: p.Address,
			Method: poolMethodInitialA,
		}, []any{&initialA}).AddCall(&ethrpc.Call{
		ABI:    metaABI,
		Target: p.Address,
		Method: poolMethodFutureA,
	}, []any{&futureA}).AddCall(&ethrpc.Call{
		ABI:    metaABI,
		Target: p.Address,
		Method: poolMethodInitialATime,
	}, []any{&initialATime}).AddCall(&ethrpc.Call{
		ABI:    metaABI,
		Target: p.Address,
		Method: poolMethodFutureATime,
	}, []any{&futureATime}).AddCall(&ethrpc.Call{
		ABI:    metaABI,
		Target: p.Address,
		Method: poolMethodFee,
	}, []any{&swapFee}).AddCall(&ethrpc.Call{
		ABI:    metaABI,
		Target: p.Address,
		Method: poolMethodAdminFee,
	}, []any{&adminFee}).AddCall(&ethrpc.Call{
		ABI:    erc20ABI,
		Target: p.GetLpToken(),
		Method: erc20MethodTotalSupply,
	}, []any{&lpSupply})

	for i := range p.Tokens {
		calls.AddCall(&ethrpc.Call{
			ABI:    metaABI,
			Target: p.Address,
			Method: poolMethodBalances,
			Params: []any{big.NewInt(int64(i))},
		}, []any{&balances[i]})
	}

	if _, err := calls.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"poolType":    p.Type,
			"error":       err,
		}).Errorf("failed to aggregate call pool data")
		return entity.Pool{}, err
	}

	var snappedRedemptionPrice *big.Int
	// Handle a specific case for the RAI Curve-Meta pool,
	// since this pool uses a different contract version, leading the "rates"
	// is calculated using contract data.
	if p.Address == RAIMetaPool {
		var redemptionPriceSnapContract common.Address
		req := d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).SetFrom(AddrDummy).
			AddCall(&ethrpc.Call{
				ABI:    metaABIV0_2_12,
				Target: p.Address,
				Method: poolMethodRedemptionPriceSnap,
			}, []any{&redemptionPriceSnapContract})

		if _, err := req.TryAggregate(); err != nil {
			logger.WithFields(logger.Fields{
				"poolAddress": p.Address,
				"poolType":    p.Type,
				"error":       err,
			}).Errorf("failed to aggregate RAI pool redemption_price_snap")
		} else {
			req = d.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).SetFrom(AddrDummy).
				AddCall(&ethrpc.Call{
					ABI:    redemptionPriceSnap,
					Target: redemptionPriceSnapContract.String(),
					Method: oracleMethodSnappedRedemptionPrice,
				}, []any{&snappedRedemptionPrice})

			if _, err := req.TryAggregate(); err != nil {
				logger.WithFields(logger.Fields{
					"poolAddress": p.Address,
					"poolType":    p.Type,
					"error":       err,
				}).Errorf("failed to aggregate RAI snappedRedemptionPrice")
			}
		}
	}

	var extra = PoolMetaExtra{
		InitialA:               initialA.String(),
		FutureA:                futureA.String(),
		InitialATime:           initialATime.Int64(),
		FutureATime:            futureATime.Int64(),
		SwapFee:                swapFee.String(),
		AdminFee:               adminFee.String(),
		SnappedRedemptionPrice: uint256.MustFromBig(snappedRedemptionPrice),
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
