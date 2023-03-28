package curve

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/big"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/pkg/logger"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/config"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	curveAave "github.com/KyberNetwork/router-service/internal/pkg/core/curve-aave"
	curveBase "github.com/KyberNetwork/router-service/internal/pkg/core/curve-base"
	curveCompound "github.com/KyberNetwork/router-service/internal/pkg/core/curve-compound"
	curveMeta "github.com/KyberNetwork/router-service/internal/pkg/core/curve-meta"
	curvePlainOracle "github.com/KyberNetwork/router-service/internal/pkg/core/curve-plain-oracle"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/core"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/curve/factory"
	"github.com/KyberNetwork/router-service/internal/pkg/scandex/uniswap"
	"github.com/KyberNetwork/router-service/internal/pkg/service"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"

	"github.com/ethereum/go-ethereum/common"
)

type Curve struct {
	properties  Properties
	scanDexCfg  *config.ScanDex
	scanService *service.ScanService
}

const (
	stringSeparator = ", "
)

func New(scanDexCfg *config.ScanDex, scanService *service.ScanService) (core.IScanDex, error) {
	properties, err := NewProperties(scanDexCfg.Properties)
	if err != nil {
		return nil, err
	}
	ctx := context.Background()

	if !factory.IsExcluded(scanDexCfg.Id) {
		var addressesFromProvider []string
		addressesFromProviderStr, err := scanService.GetCurveAddressProviders(ctx)
		if err != nil {
			logger.Errorf("failed to get config addresses from provider, err: %v", err)
			return nil, err
		}
		if len(strings.Split(addressesFromProviderStr, stringSeparator)) < 5 {
			addresses, err := factory.GetAddressesFromProvider(ctx, scanService)
			if err != nil {
				return nil, err
			} else {
				err = scanService.SetCurveAddressProviders(ctx, strings.Join(addresses, stringSeparator))
				if err != nil {
					logger.Errorf("can not save config addresses from provider, err: %v", err)
					return nil, err
				} else {
					addressesFromProvider = addresses
				}
			}
		} else {
			addressesFromProvider = strings.Split(addressesFromProviderStr, stringSeparator)
		}
		properties.AddressesFromProvider = addressesFromProvider
	}

	return &Curve{
		properties:  properties,
		scanDexCfg:  scanDexCfg,
		scanService: scanService,
	}, nil
}

// InitPool to
// - Load curve pools from data file: internal/pkg/data/chain/{chain}/curve/pools.json
// - Exclude curve pools from ignore file: internal/pkg/data/chain/{chain}/curve/pools.json.ignore
// Currently ignore pools that
// 1. tokens and underlying_token is native
// 2. IgnorePools that cannot get info from main/getter/factory SC or pool itself
func (t *Curve) InitPool(ctx context.Context) error {
	if t.properties.PoolPath == "" {
		return nil
	}

	// Load ignored pools
	ignorePoolsFile, err := os.Open(path.Join(t.scanService.Config().DataFolder, t.properties.PoolPath+".ignore"))
	if err == nil {
		byteValue, _ := io.ReadAll(ignorePoolsFile)
		var ignorePools []string
		err = json.Unmarshal(byteValue, &ignorePools)
		if err == nil {
			t.properties.IgnorePools = ignorePools
		}
	}

	// Load curve pools from data file
	poolsFile, err := os.Open(path.Join(t.scanService.Config().DataFolder, t.properties.PoolPath))
	if err != nil {
		logger.Errorf("failed to open config file: %v", err)
		return fmt.Errorf("failed to open config file: %v", err)
	}

	byteValue, err := io.ReadAll(poolsFile)
	if err != nil {
		logger.Errorf("failed to read pools.json file: %v", err)
		return fmt.Errorf("failed to read pools.json file: %v", err)
	}

	var pools []factory.PoolItem
	err = json.Unmarshal(byteValue, &pools)
	if err != nil {
		logger.Errorf("failed to parse pools: %v", err)
	}

	// Extract pools data from the data file and then storing them into the datastore
	for i := range pools {
		var pool = pools[i]

		if t.scanService.ExistPool(ctx, pool.ID) {
			continue
		}

		if len(pool.LpToken) == 0 {
			logger.Errorf("can not find lpToken of pool %v", pool.ID)
			return fmt.Errorf("can not find lpToken of pool: %v", pool.ID)
		}

		staticExtraBytes := t.ExtractStaticExtra(ctx, t.scanService, pool)
		reserves, tokens, err := t.ExtractReservesAndTokens(ctx, t.scanService, pool)
		if err != nil {
			logger.Errorf("can not extract reserves and tokens of a pool: %v", pool.ID)
			return fmt.Errorf("can not extract reserves and tokens of a pool: %v", pool.ID)
		}

		var newPool = entity.Pool{
			Address:     pool.ID,
			ReserveUsd:  0,
			SwapFee:     0,
			Exchange:    t.scanDexCfg.Id,
			Type:        pool.Type,
			Timestamp:   0,
			Reserves:    reserves,
			Tokens:      tokens,
			StaticExtra: string(staticExtraBytes),
		}
		_ = t.scanService.SavePool(ctx, newPool)
	}

	return nil
}

func (t *Curve) UpdateNewPools(ctx context.Context) {
	if len(t.properties.AddressesFromProvider) == 0 {
		logger.Errorf("There are no configuration for curve AddressProvider")
		return
	}
	mainRegistry := t.properties.AddressesFromProvider[0]
	getter := t.properties.AddressesFromProvider[1]
	metaPoolsFactory := t.properties.AddressesFromProvider[2]
	cryptoPoolsRegistry := t.properties.AddressesFromProvider[3]
	cryptoPoolsFactory := t.properties.AddressesFromProvider[4]

	offsetKeyForMainRegistryPools := utils.Join(t.scanDexCfg.Id, "offset_main_registry_pools")
	offsetKeyForMetaFactoryPool := utils.Join(t.scanDexCfg.Id, "offset_meta_factory_pool")
	offsetKeyForCryptoRegistryPool := utils.Join(t.scanDexCfg.Id, "offset_crypto_registry_pool")
	offsetKeyForCryptoFactoryPool := utils.Join(t.scanDexCfg.Id, "offset_crypto_factory_pool")

	fetchPoolFromMainRegistry := func() error {
		offset, err := t.scanService.GetLastDexOffset(ctx, offsetKeyForMainRegistryPools)
		if err != nil {
			logger.Errorf("failed to get config pair offset from database, err: %v", err)
			return err
		}
		poolAddresses, poolCount, err := factory.GetPoolAddresses(
			ctx, t.scanService, abis.CurveMainRegistry, mainRegistry, int64(offset),
		)
		if err != nil {
			return err
		}

		// Get Pool isMeta
		calls := make([]*repository.CallParams, 0, len(poolAddresses))
		isMetas := make([]bool, len(poolAddresses))
		for i := 0; i < len(poolAddresses); i++ {
			calls = append(
				calls, &repository.CallParams{
					ABI:    abis.CurveMainRegistry,
					Target: mainRegistry,
					Method: "is_meta",
					Params: []interface{}{poolAddresses[i]},
					Output: &isMetas[i],
				},
			)
		}
		if err := t.scanService.MultiCall(ctx, calls); err != nil {
			logger.Errorf("failed to process multicall, err: %v", err)
			return err
		}

		// Handle Plain and Lending Pools
		var poolPlainAndLendingAddresses []common.Address
		var poolMetaAddresses []common.Address

		for i := 0; i < len(poolAddresses); i++ {
			ignoredPoolListStr := strings.Join(t.properties.IgnorePools, stringSeparator)
			if strings.Contains(ignoredPoolListStr, strings.ToLower(poolAddresses[i].Hex())) {
				continue
			}
			if !isMetas[i] {
				poolPlainAndLendingAddresses = append(poolPlainAndLendingAddresses, poolAddresses[i])
			} else {
				poolMetaAddresses = append(poolMetaAddresses, poolAddresses[i])
			}
		}

		logger.Infof(
			"[curve-main-registry] fetching from Main Registry start (count: %v, offset: %v, total-plain: %v, total-meta: %v)....................:",
			poolCount, offset, len(poolPlainAndLendingAddresses), len(poolMetaAddresses),
		)

		var g errgroup.Group

		g.Go(
			func() error {
				return factory.CheckAndFetchPlainAndLendingPools(
					ctx,
					t.scanDexCfg.Id,
					t.scanService,
					mainRegistry,
					getter,
					poolPlainAndLendingAddresses,
				)
			},
		)

		g.Go(
			func() error {
				return factory.CheckAndFetchMetaPools(
					ctx,
					t.scanDexCfg.Id,
					t.scanService,
					metaPoolsFactory,
					poolMetaAddresses,
				)
			},
		)

		err = g.Wait()
		if err != nil {
			logger.Errorf("failed to fetch Curve pools err: %v", err)
			return err
		}

		err = t.scanService.SetLastDexOffset(ctx, offsetKeyForMainRegistryPools, poolCount)
		if err != nil {
			logger.Errorf("can not save config pair offset to database err %v", err)
			return err
		}

		logger.Infoln("[curve-main-registry] fetching from Main Registry end....................:")
		return nil
	}

	fetchPoolsFromMetaFactory := func() error {
		offset, err := t.scanService.GetLastDexOffset(ctx, offsetKeyForMetaFactoryPool)
		if err != nil {
			logger.Errorf("failed to get config pair offset from database, err: %v", err)
			return err
		}

		poolAddresses, poolCount, err := factory.GetPoolAddresses(
			ctx, t.scanService, abis.CurveMetaFactory, metaPoolsFactory, int64(offset),
		)
		if err != nil {
			return err
		}

		logger.Infof(
			"[curve-meta-factory] fetching from Meta Factory start(count: %v, offset: %v, total: %v)....................:",
			poolCount, offset, len(poolAddresses),
		)
		err = factory.CheckAndFetchMetaPools(
			ctx,
			t.scanDexCfg.Id,
			t.scanService,
			metaPoolsFactory,
			poolAddresses,
		)
		if err == nil {
			err = t.scanService.SetLastDexOffset(ctx, offsetKeyForMetaFactoryPool, poolCount)
			if err != nil {
				logger.Errorf("can not save config pair offset to database err %v", err)
				return err
			}
		}

		logger.Infof("[curve-meta-factory] fetching from Meta Factory end....................")
		return nil
	}

	fetchCryptoPoolsFromRegistryAndFactory := func() error {
		cryptoRegistryOffset, err := t.scanService.GetLastDexOffset(ctx, offsetKeyForCryptoRegistryPool)
		if err != nil {
			logger.Errorf("failed to get config pair offset from database, err: %v", err)
			return err
		}
		cryptoFactoryPoolOffset, err := t.scanService.GetLastDexOffset(ctx, offsetKeyForCryptoFactoryPool)
		if err != nil {
			logger.Errorf("failed to get config pair offset from database, err: %v", err)
			return err
		}

		cryptoSwapFactory := factory.New(t)

		registryPoolAddresses, registryPoolCount, err := factory.GetPoolAddresses(
			ctx, t.scanService, abis.CurveCryptoRegistry, cryptoPoolsRegistry, int64(cryptoRegistryOffset),
		)
		if err != nil {
			return err
		}

		factoryPoolAddresses, factoryPoolCount, err := factory.GetPoolAddresses(
			ctx, t.scanService, abis.CurveCryptoFactory, cryptoPoolsFactory, int64(cryptoFactoryPoolOffset),
		)
		if err != nil {
			return err
		}

		poolAddresses := append(registryPoolAddresses, factoryPoolAddresses...)
		poolCount := registryPoolCount + factoryPoolCount

		logger.Infof(
			"[curve-crypto] fetching from Crypto Factory start(count: %v, cryptoRegistryOffset: %v, cryptoFactoryPoolOffset: %v, total: %v)....................:",
			poolCount, cryptoRegistryOffset, cryptoFactoryPoolOffset, len(poolAddresses),
		)

		// There are no other pool types other than the two and tricrypto pool at the moment
		twoPoolsFromRegistry, tricryptoPoolsFromRegistry, _, err := cryptoSwapFactory.FetchPoolFromRegistry(
			ctx, t.scanService, cryptoPoolsRegistry, registryPoolAddresses,
		)
		if err != nil {
			logger.Errorf("failed to fetch pools from curve-crypto-registry, err: %v", err)
		}
		// There are only two pools from crypto-factory
		twoPoolsFromFactory, err := cryptoSwapFactory.FetchPoolFromFactory(
			ctx, t.scanService, cryptoPoolsFactory, factoryPoolAddresses,
		)
		if err != nil {
			logger.Errorf("failed to fetch pools from curve-crypto-factory, err: %v", err)
		}

		twoPools := append(twoPoolsFromRegistry, twoPoolsFromFactory...)
		tricryptoPools := tricryptoPoolsFromRegistry

		err = cryptoSwapFactory.AddCryptoPools(
			ctx, t.scanDexCfg.Id, t.scanService, constant.PoolTypes.CurveTwo, abis.CurveTwo, twoPools,
		)
		if err != nil {
			logger.Errorf("failed to add curve-two pools, err: %v", err)
		}
		err = cryptoSwapFactory.AddCryptoPools(
			ctx, t.scanDexCfg.Id, t.scanService, constant.PoolTypes.CurveTricrypto, abis.CurveTricrypto, tricryptoPools,
		)
		if err != nil {
			logger.Errorf("failed to add curve-tricrypto pools, err: %v", err)
		}
		if err == nil {
			err1 := t.scanService.SetLastDexOffset(ctx, offsetKeyForCryptoRegistryPool, registryPoolCount)
			err2 := t.scanService.SetLastDexOffset(ctx, offsetKeyForCryptoFactoryPool, factoryPoolCount)
			if err1 != nil || err2 != nil {
				logger.Errorf("can not save config pair offset to database err %v", err)
				return err
			}
		}

		logger.Infof("[curve-crypto] fetching from Crypto Factory end....................")
		return nil
	}

	for {
		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			err := fetchPoolFromMainRegistry()
			if err != nil {
				logger.Errorf("can not update new pool (main registry) %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			err := fetchPoolsFromMetaFactory()
			if err != nil {
				logger.Errorf("can not update new pool (meta factory) %v", err)
			}
		}()
		go func() {
			defer wg.Done()
			err := fetchCryptoPoolsFromRegistryAndFactory()
			if err != nil {
				logger.Errorf("can not update new pool (meta factory) %v", err)
			}
		}()
		wg.Wait()
		time.Sleep(time.Duration(t.properties.NewPoolJobIntervalSec) * time.Second)
	}
}

func (t *Curve) UpdateReserves(ctx context.Context) {
	f := func(ctx context.Context, scanService *service.ScanService, pools []entity.Pool) int {

		noReserveCount := 0
		for _, pool := range pools {
			var reserves []*big.Int
			var extra interface{}
			switch pool.Type {
			case constant.PoolTypes.CurveAave:
				{
					reserves, _ = t.getAavePoolReserves(ctx, pool)
					extra, _ = t.getAavePoolExtra(ctx, pool)
				}
			case constant.PoolTypes.CurveBase:
				{
					reserves, _ = t.getBasePoolReserves(ctx, pool)
					extra, _ = t.getBasePoolExtra(ctx, pool)
				}
			case constant.PoolTypes.CurvePlainOracle:
				{
					reserves, _ = t.getPlainOraclePoolReserves(ctx, pool)
					extra, _ = t.getPlainOraclePoolExtra(ctx, pool)
				}
			case constant.PoolTypes.CurveTwo:
				{
					extra, reserves, _ = t.getTwoPoolData(ctx, pool)
				}
			case constant.PoolTypes.CurveTricrypto:
				{
					extra, reserves, _ = t.getTricryptoPoolData(ctx, pool)
				}
			case constant.PoolTypes.CurveMeta:
				{
					extra, reserves, _ = t.getMetaPoolData(ctx, pool)
				}
			case constant.PoolTypes.CurveCompound:
				{
					reserves, _ = t.getCompoundPoolReserves(ctx, pool)
					extra, _ = t.getCompoundExtra(ctx, pool)
				}
			default:
				reserves = nil
			}

			if extra != nil {
				extraBytes, err := json.Marshal(extra)
				if err != nil {
					logger.Errorf("Fail to marshal extra data: %v", extra)
				}
				_ = t.scanService.UpdatePoolExtra(ctx, pool.Address, string(extraBytes))
			}
			if reserves != nil {
				reservesStr := make([]string, len(reserves))
				if !strings.Contains(
					strings.Join(t.properties.IgnorePools, stringSeparator), strings.ToLower(pool.Address),
				) {
					for j := range reserves {
						reservesStr[j] = reserves[j].String()
					}
				} else {
					for j := range reserves {
						reservesStr[j] = "0"
					}
				}
				_ = t.scanService.UpdatePoolReserve(ctx, pool.Address, time.Now().Unix(), reservesStr)
			} else {
				noReserveCount = noReserveCount + 1
			}
		}
		return len(pools) - noReserveCount
	}

	uniswap.UpdateReserveJob(
		ctx,
		t.scanDexCfg,
		t.scanService,
		f,
		t.properties.ReserveJobInterval,
		t.properties.UpdateReserveBulk,
		t.properties.ConcurrentBatches,
	)
}

func (t *Curve) UpdateTotalSupply(ctx context.Context) {
	uniswap.UpdateTotalSupplyJob(
		ctx,
		t.scanDexCfg,
		t.scanService,
		uniswap.UpdateTotalSupplyHandler,
		t.properties.TotalSupplyJobIntervalSec,
		t.properties.UpdateReserveBulk,
	)
}

func (t *Curve) ExtractStaticExtra(
	ctx context.Context, scanService *service.ScanService, poolItem factory.PoolItem,
) (staticExtraBytes []byte) {
	if poolItem.Type == constant.PoolTypes.CurveBase || poolItem.Type == constant.PoolTypes.CurveTwo || poolItem.Type == constant.PoolTypes.CurveTricrypto {
		var staticExtra = curveBase.PoolStaticExtra{
			LpToken:    poolItem.LpToken,
			APrecision: poolItem.APrecision,
		}
		for j := range poolItem.Tokens {
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
			staticExtra.Rates = append(staticExtra.Rates, poolItem.Tokens[j].Rate)
		}
		staticExtraBytes, _ = json.Marshal(staticExtra)
	} else if poolItem.Type == constant.PoolTypes.CurvePlainOracle {
		var staticExtra = curvePlainOracle.PoolStaticExtra{
			LpToken:    poolItem.LpToken,
			APrecision: poolItem.APrecision,
		}
		for j := range poolItem.Tokens {
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
		}
		staticExtraBytes, _ = json.Marshal(staticExtra)
	} else if poolItem.Type == constant.PoolTypes.CurveAave {
		var staticExtra = curveAave.PoolStaticExtra{
			LpToken:          poolItem.LpToken,
			UnderlyingTokens: poolItem.UnderlyingTokens,
		}
		for j := range poolItem.Tokens {
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
		}
		staticExtraBytes, _ = json.Marshal(staticExtra)
	} else if poolItem.Type == constant.PoolTypes.CurveCompound {
		var staticExtra = curveCompound.PoolStaticExtra{
			LpToken:          poolItem.LpToken,
			UnderlyingTokens: poolItem.UnderlyingTokens,
		}
		for j := range poolItem.Tokens {
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
		}
		staticExtraBytes, _ = json.Marshal(staticExtra)
	} else if poolItem.Type == constant.PoolTypes.CurveMeta {
		var staticExtra = curveMeta.PoolStaticExtra{
			LpToken:          poolItem.LpToken,
			BasePool:         poolItem.BasePool,
			RateMultiplier:   poolItem.RateMultiplier,
			APrecision:       poolItem.APrecision,
			UnderlyingTokens: poolItem.UnderlyingTokens,
		}
		for j := range poolItem.Tokens {
			staticExtra.PrecisionMultipliers = append(staticExtra.PrecisionMultipliers, poolItem.Tokens[j].Precision)
			staticExtra.Rates = append(staticExtra.Rates, poolItem.Tokens[j].Rate)
		}
		staticExtraBytes, _ = json.Marshal(staticExtra)
	}

	return staticExtraBytes
}

func (t *Curve) ExtractReservesAndTokens(
	ctx context.Context, scanService *service.ScanService, poolItem factory.PoolItem,
) (reserves entity.PoolReserves, tokens []*entity.PoolToken, err error) {
	reserves = make(entity.PoolReserves, 1)
	tokens = make([]*entity.PoolToken, 0)

	reserves[0] = "0"
	for j := range poolItem.Tokens {
		// check token exists
		if poolItem.Type == constant.PoolTypes.CurveAave {
			if _, err := scanService.FetchOrGetTokenType(
				ctx, poolItem.Tokens[j].Address, "aave", poolItem.UnderlyingTokens[j],
			); err != nil {
				return nil, nil, err
			}
			tokens = append(
				tokens, &entity.PoolToken{
					Address:   poolItem.Tokens[j].Address,
					Weight:    1,
					Swappable: false,
				},
			)
		} else {
			if _, err := scanService.FetchOrGetToken(ctx, poolItem.Tokens[j].Address); err != nil {
				return nil, nil, err
			}
			tokens = append(
				tokens, &entity.PoolToken{
					Address:   poolItem.Tokens[j].Address,
					Weight:    1,
					Swappable: true,
				},
			)
		}

		reserves = append(reserves, "0")
	}

	return reserves, tokens, nil
}
