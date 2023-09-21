package synthetix

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/timer"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	cfg                                   *Config
	poolStateReader                       IPoolStateReader
	systemSettingsReader                  ISystemSettingsReader
	exchangerWithFeeRecAlternativesReader IExchangerWithFeeRecAlternativesReader
	exchangeRatesReader                   IExchangeRatesReader
	chainlinkDataFeedReader               IChainlinkDataFeedReader
	dexPriceAggregatorUniswapV3Reader     IDexPriceAggregatorUniswapV3Reader
}

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	poolStateVersion := getPoolStateVersion(valueobject.ChainID(cfg.ChainID))

	if poolStateVersion == PoolStateVersionNormal {
		return &PoolTracker{
			cfg:                     cfg,
			poolStateReader:         NewPoolStateReader(cfg, ethrpcClient),
			systemSettingsReader:    NewSystemSettingsReader(cfg, ethrpcClient),
			exchangeRatesReader:     NewExchangeRatesReader(cfg, ethrpcClient),
			chainlinkDataFeedReader: NewChainlinkDataFeedReader(cfg, ethrpcClient),
		}
	}

	return &PoolTracker{
		cfg:                                   cfg,
		poolStateReader:                       NewPoolStateReader(cfg, ethrpcClient),
		systemSettingsReader:                  NewSystemSettingsReader(cfg, ethrpcClient),
		exchangerWithFeeRecAlternativesReader: NewExchangerWithFeeRecAlternativesReader(cfg, ethrpcClient),
		exchangeRatesReader:                   NewExchangeRatesWithDexPricingReader(cfg, ethrpcClient),
		chainlinkDataFeedReader:               NewChainlinkDataFeedReader(cfg, ethrpcClient),
		dexPriceAggregatorUniswapV3Reader:     NewDexPriceAggregatorUniswapV3Reader(cfg, ethrpcClient),
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	pool entity.Pool,
	_ sourcePool.GetNewPoolStateParams,
) (entity.Pool, error) {
	finish := timer.Start(fmt.Sprintf("[%s] get new pool state", d.cfg.DexID))
	defer finish()

	poolState, err := d.getPoolState(ctx, pool.Address)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pool state")
		return entity.Pool{}, err
	}

	address := d.cfg.Addresses
	poolState.Addresses = &address

	systemSettings, err := d.getSystemSettings(ctx, poolState)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get system settings")
		return entity.Pool{}, err
	}
	poolState.SystemSettings = systemSettings

	poolState, err = d.getExchangerWithFeeRecAlternativesData(ctx, poolState)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get exchanger with fee rec alternatives data")
		return entity.Pool{}, err
	}

	poolState, err = d.getExchangeRates(ctx, poolState)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get exchange rates")
		return entity.Pool{}, err
	}

	chainlinkNumRounds := d.getChainlinkNumRounds(poolState.SystemSettings.DynamicFeeConfig.Rounds)

	aggregators, err := d.getChainlinkDataFeeds(
		ctx,
		poolState.SUSDCurrencyKey,
		poolState.AggregatorAddresses,
		chainlinkNumRounds,
	)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get chainlink data feeds")
		return entity.Pool{}, err
	}
	poolState.Aggregators = aggregators

	dexPriceAggregatorUniswapV3, err := d.getDexPriceAggregatorUniswapV3(ctx, poolState)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get dex price aggregator UniswapV3")
		return entity.Pool{}, err
	}
	poolState.DexPriceAggregator = dexPriceAggregatorUniswapV3

	p, err := d.newPool(poolState.Addresses.Synthetix, poolState)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not initialze new pool")
		return entity.Pool{}, err
	}

	return *p, nil
}

// ================================================================================

func (d *PoolTracker) newPool(address string, poolState *PoolState) (*entity.Pool, error) {
	var (
		poolTokens = make([]*entity.PoolToken, 0, len(poolState.Synths))
		reserves   = make(entity.PoolReserves, 0, len(poolState.Synths))
	)

	for _, currencyKey := range poolState.CurrencyKeys {
		synthAddress := poolState.Synths[currencyKey]
		poolTokens = append(poolTokens, &entity.PoolToken{
			Address:   strings.ToLower(synthAddress.String()),
			Swappable: true,
		})
		reserves = append(reserves, poolState.SynthsTotalSupply[currencyKey].String())
	}

	extra := Extra{PoolState: poolState}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not marshal extra")
		return nil, err
	}

	return &entity.Pool{
		Address:   strings.ToLower(address),
		Exchange:  d.cfg.DexID,
		Type:      DexTypeSynthetix,
		Timestamp: time.Now().Unix(),
		Tokens:    poolTokens,
		Reserves:  reserves,
		Extra:     string(extraBytes),
	}, nil
}

func (d *PoolTracker) getDexPriceAggregatorUniswapV3(ctx context.Context, poolState *PoolState) (*DexPriceAggregatorUniswapV3, error) {
	poolStateVersion := getPoolStateVersion(valueobject.ChainID(d.cfg.ChainID))

	// Normal version does not have DexPriceAggregatorUniswapV3
	if poolStateVersion == PoolStateVersionNormal {
		return nil, nil
	}

	dexPriceAggregatorUniswapV3, err := d.dexPriceAggregatorUniswapV3Reader.Read(ctx, poolState)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get dex price aggregator UniswapV3")
		return nil, err
	}

	dexPriceAggregatorUniswapV3.BlockTimestamp = poolState.BlockTimestamp

	return dexPriceAggregatorUniswapV3, nil
}

func (d *PoolTracker) getChainlinkDataFeeds(
	ctx context.Context,
	sUSDCurrencyKey string,
	aggregatorAddresses map[string]common.Address,
	numRound *big.Int,
) (map[string]*ChainlinkDataFeed, error) {
	var (
		roundCount  = int(numRound.Int64())
		aggregators = make(map[string]*ChainlinkDataFeed, len(aggregatorAddresses))
	)

	for aggregatorKey, aggregatorAddress := range aggregatorAddresses {
		if aggregatorKey == sUSDCurrencyKey {
			continue
		}

		aggregator, err := d.chainlinkDataFeedReader.Read(ctx, aggregatorAddress.String(), roundCount)
		if err != nil {
			logger.WithFields(logger.Fields{
				"dexID": d.cfg.DexID,
				"error": err,
			}).Error("can not get chainlink data feeds")
			return nil, err
		}

		aggregators[aggregatorKey] = aggregator
	}

	return aggregators, nil
}

func (d *PoolTracker) getChainlinkNumRounds(dynamicFeeConfigRounds *big.Int) *big.Int {
	var chainlinkNumRounds *big.Int
	// Choose bigger number of num rounds because we need to get all required data
	if dynamicFeeConfigRounds.Cmp(DefaultChainlinkNumRounds) > 0 {
		chainlinkNumRounds = dynamicFeeConfigRounds
	} else {
		chainlinkNumRounds = DefaultChainlinkNumRounds
	}

	return chainlinkNumRounds
}

func (d *PoolTracker) getExchangeRates(ctx context.Context, poolState *PoolState) (*PoolState, error) {
	poolState, err := d.exchangeRatesReader.Read(ctx, poolState)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get exchange rates")
		return nil, err
	}

	return poolState, nil

}

func (d *PoolTracker) getExchangerWithFeeRecAlternativesData(ctx context.Context, poolState *PoolState) (*PoolState, error) {
	poolStateVersion := getPoolStateVersion(valueobject.ChainID(d.cfg.ChainID))

	// Normal version does not have to fetch this data
	if poolStateVersion == PoolStateVersionNormal {
		return poolState, nil
	}

	poolState, err := d.exchangerWithFeeRecAlternativesReader.Read(ctx, poolState)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get exchanger with fee rec alternatives data")
		return nil, err
	}

	return poolState, nil

}

func (d *PoolTracker) getSystemSettings(ctx context.Context, poolState *PoolState) (*SystemSettings, error) {
	systemSettings, err := d.systemSettingsReader.Read(ctx, poolState)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get system settings")
		return nil, err
	}

	return systemSettings, nil
}

func (d *PoolTracker) getPoolState(ctx context.Context, address string) (*PoolState, error) {
	poolState, err := d.poolStateReader.Read(ctx, address)
	if err != nil {
		logger.WithFields(logger.Fields{
			"dexID": d.cfg.DexID,
			"error": err,
		}).Error("can not get pool state")
		return nil, err
	}

	return poolState, nil
}
