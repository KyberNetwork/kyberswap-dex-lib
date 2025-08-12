package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/KyberNetwork/kutils/klog"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dexalot"
	"github.com/goccy/go-json"
	"github.com/mcuadros/go-defaults"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"github.com/samber/lo"
	"github.com/spf13/viper"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/buildroute"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/util/env"
)

const (
	EmptyConfigHash = ""
)

type IRemoteConfigFetcher interface {
	Fetch(ctx context.Context) (valueobject.RemoteConfig, error)
}

type ConfigLoader struct {
	path                string
	additionConfigPaths []string
	config              *Config
	mu                  sync.RWMutex
	remoteConfigFetcher IRemoteConfigFetcher
}

// NewConfigLoader returns a new ConfigLoader.
func NewConfigLoader(
	path string,
	additionConfigPaths []string,
) (*ConfigLoader, error) {
	cl := &ConfigLoader{
		path:                path,
		additionConfigPaths: additionConfigPaths,
	}
	err := cl.Initialize()
	if err != nil {
		return nil, err
	}

	return cl, nil
}

func (cl *ConfigLoader) SetRemoteConfigFetcher(
	remoteConfigFetcher IRemoteConfigFetcher,
) {
	cl.remoteConfigFetcher = remoteConfigFetcher
}

func (cl *ConfigLoader) GetLocalConfig() (*Config, error) {
	viper.SetConfigFile(cl.path)

	c := &Config{}
	defaults.SetDefaults(c)

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Read config file failed. ", err)

		configBuffer, err := json.Marshal(c)

		if err != nil {
			return nil, err
		}

		err = viper.ReadConfig(bytes.NewBuffer(configBuffer))
		if err != nil {
			return nil, err
		}
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "__", "-", "_"))
	viper.AutomaticEnv()
	viper.AllowEmptyEnv(true)

	// Load and merge additional custom path configuration
	for _, configPath := range cl.additionConfigPaths {
		configViper := viper.New()
		if configPath != "" {
			configViper.SetConfigFile(configPath)
			if err := configViper.ReadInConfig(); err != nil {
				return nil, fmt.Errorf("failed to read config path: %s, err: %w", configPath, err)
			}
			_ = viper.MergeConfigMap(configViper.AllSettings())
		}
	}

	decoder := mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
		StringToBigIntHookFunc(),
	)
	decodeConfigOption := viper.DecodeHook(decoder)
	if err := viper.Unmarshal(c, decodeConfigOption); err != nil {
		fmt.Printf("failed to unmarshal config %v\n", err)
		return nil, err
	}

	// set default gRPC config
	if c.GRPC.Host == "" && c.GRPC.Port == 0 {
		c.GRPC = ServerListen{
			Port: 10443,
			Host: "0.0.0.0",
		}
	}
	c.UseCase.GetRoute.GasTokenAddress = strings.ToLower(valueobject.WrappedNativeMap[c.Common.ChainID])
	c.UseCase.PoolFactory.UseAEVM = c.AEVMEnabled
	c.UseCase.PoolManager.FeatureFlags.IsAEVMEnabled = c.AEVMEnabled
	c.UseCase.TradeDataGenerator.UseAEVM = c.AEVMEnabled
	c.UseCase.BuildRoute.TokenGroups = c.UseCase.GetRoute.SafetyQuoteConfig.TokenGroupConfig
	fmt.Println("ENV:", viper.GetString("ENV"))
	fmt.Println("GOMAXPROCS:", runtime.GOMAXPROCS(0))

	return c, nil
}

// GetRemoteConfig gets the config from ks-settings API
func (cl *ConfigLoader) GetRemoteConfig(ctx context.Context) (valueobject.RemoteConfig, error) {
	remoteCfg, err := cl.remoteConfigFetcher.Fetch(ctx)
	if err != nil {
		return valueobject.RemoteConfig{}, err
	}

	return remoteCfg, nil
}

// Initialize sets the local config (default + file)
func (cl *ConfigLoader) Initialize() error {
	cfg, err := cl.GetLocalConfig()
	if err != nil {
		return err
	}

	// Set config
	cl.mu.Lock()
	cl.config = cfg
	cl.mu.Unlock()

	_, _ = klog.InitLogger(cfg.Log.Configuration, klog.LoggerBackendZap)
	if env.IsLocalMode() {
		log.Logger = zerolog.New(zerolog.NewConsoleWriter())
	} else {
		log.Logger = zerolog.New(os.Stdout)
	}
	log.Logger = log.Logger.Level(parseLevel(cfg.Log.ConsoleLevel)).With().Timestamp().Caller().Logger()
	zerolog.CallerMarshalFunc = func(pc uintptr, file string, line int) string {
		return filepath.Base(filepath.Dir(file)) + "/" + filepath.Base(file) + ":" + strconv.Itoa(line)
	}
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
	zerolog.InterfaceMarshalFunc = json.MarshalNoEscape
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnixMs
	zerolog.DefaultContextLogger = &log.Logger

	log.Info().Any("local config", cl.config).Send()
	return nil
}

func parseLevel(levelStr string) zerolog.Level {
	level, err := zerolog.ParseLevel(levelStr)
	if err != nil {
		level = zerolog.InfoLevel
	}
	return level
}

func (cl *ConfigLoader) Reload(ctx context.Context) error {
	remoteCfg, err := cl.GetRemoteConfig(ctx)
	if err != nil {
		log.Ctx(ctx).Err(err).Msg("failed to fetch config from remote")
		return err
	}

	// Only override the config when remote config hash is NOT empty
	if remoteCfg.Hash != EmptyConfigHash {
		// Set config
		cl.mu.Lock()

		// Set each field so that it does not override the address of the pointer cl.config
		cl.setAvailableSources(remoteCfg.AvailableSources)
		cl.setUnscalableSources(remoteCfg.UnscalableSources)
		cl.setExcludedSourcesByClient(remoteCfg.ExcludedSourcesByClient)
		cl.setForcePoolsForTokenByClient(remoteCfg.ForcePoolsForTokenByClient)
		cl.setValidateChecksumBySource(remoteCfg.ValidateChecksumBySource)
		cl.setDexUseAEVM(remoteCfg.DexUseAEVM)
		cl.setWhitelistedTokens(remoteCfg.WhitelistedTokens)
		cl.setBlacklistedPools(remoteCfg.BlacklistedPools)
		cl.setLog(remoteCfg.Log)
		cl.setFinderOptions(remoteCfg.FinderOptions)
		cl.setGetBestPoolOptions(remoteCfg.GetBestPoolsOptions)
		cl.setCacheConfig(remoteCfg.CacheConfig)
		cl.setBlacklistedRecipients(remoteCfg.BlacklistedRecipients)
		cl.setFaultyPoolsConfig(remoteCfg.FaultyPoolsConfig)
		cl.setSafetyQuoteReduction(remoteCfg.SafetyQuoteReduction)
		cl.setPoolManagerOptionsFromFinderOptions(remoteCfg.FinderOptions)
		cl.setRFQAcceptableSlippageFraction(remoteCfg.RFQAcceptableSlippageFraction)
		cl.setDexalotUpscalePercent(remoteCfg.DexalotUpscalePercent)
		cl.setAlphaFeeConfig(remoteCfg.AlphaFeeConfig)
		cl.setScaleHelperClients(remoteCfg.ScaleHelperClients)
		cl.setWhitelistedPrices(remoteCfg.WhitelistedPrices)
		cl.setFeatureFlags(remoteCfg.FeatureFlags)
		cl.mu.Unlock()
	}

	prettyJsonCfg, err := json.Marshal(cl.config)
	if err != nil {
		return err
	}

	fmt.Printf("config: %+v\n", string(prettyJsonCfg))

	return nil
}

func (cl *ConfigLoader) Get() (*Config, error) {
	cl.mu.RLock()
	defer cl.mu.RUnlock()
	return cl.config, nil
}

func (cl *ConfigLoader) setAvailableSources(availableSources []valueobject.Source) {
	strAvailableSources := make([]string, 0, len(availableSources))

	for _, s := range availableSources {
		strAvailableSources = append(strAvailableSources, string(s))
	}

	cl.config.Common.AvailableSources = strAvailableSources
	cl.config.UseCase.GetCustomRoute.AvailableSources = strAvailableSources
	cl.config.UseCase.GetRoute.AvailableSources = strAvailableSources
	cl.config.UseCase.TradeDataGenerator.AvailableSources = strAvailableSources
	cl.config.UseCase.PoolManager.AvailableSources = strAvailableSources
}

func (cl *ConfigLoader) setUnscalableSources(unscalableSources []valueobject.Source) {
	strUnscalableSources := make([]string, 0, len(unscalableSources))

	for _, s := range unscalableSources {
		strUnscalableSources = append(strUnscalableSources, string(s))
	}

	cl.config.UseCase.GetCustomRoute.UnscalableSources = strUnscalableSources
	cl.config.UseCase.GetRoute.UnscalableSources = strUnscalableSources
}

func (cl *ConfigLoader) setExcludedSourcesByClient(sourcesByClient map[string][]valueobject.Source) {
	newSourcesByClient := make(map[string][]string, len(sourcesByClient))

	for client, sources := range sourcesByClient {
		strSources := make([]string, 0, len(sources))
		for _, source := range sources {
			strSources = append(strSources, string(source))
		}
		newSourcesByClient[client] = strSources
	}

	cl.config.Common.ExcludedSourcesByClient = newSourcesByClient
	cl.config.UseCase.GetCustomRoute.ExcludedSourcesByClient = newSourcesByClient
	cl.config.UseCase.GetRoute.ExcludedSourcesByClient = newSourcesByClient
}

func (cl *ConfigLoader) setForcePoolsForTokenByClient(forcePoolsForTokenByClient map[string]map[string][]string) {
	for _, forcePoolsForToken := range forcePoolsForTokenByClient {
		for token, pools := range forcePoolsForToken {
			delete(forcePoolsForToken, token)
			forcePoolsForToken[strings.ToLower(token)] = lo.Map(pools, func(pool string, _ int) string {
				return strings.ToLower(pool)
			})
		}
	}
	cl.config.UseCase.GetCustomRoute.ForcePoolsForTokenByClient = forcePoolsForTokenByClient
	cl.config.UseCase.GetRoute.ForcePoolsForTokenByClient = forcePoolsForTokenByClient
}

func (cl *ConfigLoader) setValidateChecksumBySource(validateChecksumBySource map[string]bool) {
	for source, validateChecksum := range validateChecksumBySource {
		cl.config.UseCase.BuildRoute.ValidateChecksumBySource[source] = validateChecksum
	}
}

func (cl *ConfigLoader) setDexUseAEVM(dexUseAEVM map[string]bool) {
	if cl.config.UseCase.PoolFactory.DexUseAEVM == nil {
		cl.config.UseCase.PoolFactory.DexUseAEVM = make(map[string]bool)
	}
	if cl.config.UseCase.GetRoute.Aggregator.DexUseAEVM == nil {
		cl.config.UseCase.GetRoute.Aggregator.DexUseAEVM = make(map[string]bool)
	}
	if cl.config.UseCase.TradeDataGenerator.DexUseAEVM == nil {
		cl.config.UseCase.TradeDataGenerator.DexUseAEVM = make(map[string]bool)
	}
	for dex, useAEVM := range dexUseAEVM {
		cl.config.UseCase.PoolFactory.DexUseAEVM[dex] = useAEVM
		cl.config.UseCase.GetRoute.Aggregator.DexUseAEVM[dex] = useAEVM
		cl.config.UseCase.TradeDataGenerator.DexUseAEVM[dex] = useAEVM
	}
}

func (cl *ConfigLoader) setWhitelistedTokens(whitelistedTokens []valueobject.WhitelistedToken) {
	whitelistedTokenSet := make(map[string]bool, len(whitelistedTokens))
	for _, whitelistedToken := range whitelistedTokens {
		whitelistedTokenSet[strings.ToLower(whitelistedToken.Address)] = true
	}

	cl.config.Common.WhitelistedTokenSet = whitelistedTokenSet
	cl.config.UseCase.GetCustomRoute.Aggregator.WhitelistedTokenSet = whitelistedTokenSet
	cl.config.UseCase.GetRoute.Aggregator.WhitelistedTokenSet = whitelistedTokenSet
	cl.config.UseCase.BuildRoute.FaultyPoolsConfig.WhitelistedTokenSet = whitelistedTokenSet
	cl.config.UseCase.IndexPools.WhitelistedTokenSet = whitelistedTokenSet
	cl.config.UseCase.TradeDataGenerator.WhitelistedTokenSet = whitelistedTokenSet
	cl.config.Repository.Token.GoCache.WhitelistedTokenSet = whitelistedTokenSet
}

func (cl *ConfigLoader) setBlacklistedPools(blacklistedPools []string) {
	blacklistedPoolSet := make(map[string]bool, len(blacklistedPools))
	for _, blacklistedPool := range blacklistedPools {
		blacklistedPoolSet[strings.ToLower(blacklistedPool)] = true
	}

	cl.config.Common.BlacklistedPoolsSet = blacklistedPoolSet
	cl.config.UseCase.PoolManager.BlacklistedPoolSet = blacklistedPoolSet
	cl.config.UseCase.TradeDataGenerator.BlacklistedPoolSet = blacklistedPoolSet
}

func (cl *ConfigLoader) setFeatureFlags(featureFlags valueobject.FeatureFlags) {
	featureFlags.IsAEVMEnabled = featureFlags.IsAEVMEnabled && cl.config.AEVMEnabled
	cl.config.Common.FeatureFlags = featureFlags
	cl.config.UseCase.GetCustomRoute.Aggregator.FeatureFlags = featureFlags
	cl.config.UseCase.GetRoute.FeatureFlags = featureFlags
	cl.config.UseCase.GetRoute.Aggregator.FeatureFlags = featureFlags
	cl.config.UseCase.GetRoute.Cache.FeatureFlags = featureFlags
	cl.config.UseCase.BuildRoute.FeatureFlags = featureFlags
	cl.config.Validator.BuildRouteParams.FeatureFlags = featureFlags
	cl.config.Validator.GetRouteEncodeParams.FeatureFlags = featureFlags
	cl.config.UseCase.PoolFactory.UseAEVM = featureFlags.IsAEVMEnabled || featureFlags.IsRPCPoolEnabled
	cl.config.UseCase.PoolManager.FeatureFlags = featureFlags
	cl.config.UseCase.TradeDataGenerator.UseAEVM = (featureFlags.IsAEVMEnabled || featureFlags.IsRPCPoolEnabled) && !featureFlags.IgnoreAEVM
}

func (cl *ConfigLoader) setLog(logCfg valueobject.Log) {
	if logCfg.ConsoleLevel == "" {
		return
	}
	cl.config.Log.Configuration.ConsoleLevel = logCfg.ConsoleLevel
	_ = klog.SetLogLevel(context.Background(), logCfg.ConsoleLevel)
	log.Logger = log.Logger.Level(parseLevel(logCfg.ConsoleLevel))
}

func (cl *ConfigLoader) setFinderOptions(finderOptions valueobject.FinderOptions) {
	cl.config.UseCase.GetCustomRoute.Aggregator.FinderOptions = finderOptions
	cl.config.UseCase.GetRoute.Aggregator.FinderOptions = finderOptions
}

func (cl *ConfigLoader) setPoolManagerOptionsFromFinderOptions(finderOptions valueobject.FinderOptions) {
	cl.config.UseCase.PoolManager.UseAEVMRemoteFinder = finderOptions.UseAEVMRemoteFinder
}

func (cl *ConfigLoader) setGetBestPoolOptions(getBestPoolsOptions valueobject.GetBestPoolsOptions) {
	cl.config.UseCase.GetCustomRoute.Aggregator.GetBestPoolsOptions = getBestPoolsOptions
	cl.config.UseCase.GetRoute.Aggregator.GetBestPoolsOptions = getBestPoolsOptions
	cl.config.UseCase.UpdateLiquidityScoreConfig.GetBestPoolsOptions = getBestPoolsOptions
	cl.config.UseCase.GetCustomRoute.Aggregator.GetBestPoolsOptions = getBestPoolsOptions
}

func (cl *ConfigLoader) setCacheConfig(cacheConfig valueobject.CacheConfig) {
	cl.config.UseCase.GetRoute.Cache = cacheConfig
}

func (cl *ConfigLoader) setBlacklistedRecipients(blacklistedRecipients []string) {
	blacklistedRecipientSet := make(map[string]bool, len(blacklistedRecipients))
	for _, blacklistedRecipient := range blacklistedRecipients {
		blacklistedRecipientSet[strings.ToLower(blacklistedRecipient)] = true
	}

	cl.config.Validator.BuildRouteParams.BlacklistedRecipientSet = blacklistedRecipientSet
	cl.config.Validator.GetRouteEncodeParams.BlacklistedRecipientSet = blacklistedRecipientSet
}

func (cl *ConfigLoader) setFaultyPoolsConfig(faultyPoolsConfig valueobject.FaultyPoolsConfig) {
	slippageConfigByGroup := make(map[string]buildroute.SlippageGroupConfig,
		len(faultyPoolsConfig.SlippageConfigByGroup))
	for group, config := range faultyPoolsConfig.SlippageConfigByGroup {
		slippageConfigByGroup[group] = buildroute.SlippageGroupConfig{
			Buffer:       config.Buffer,
			MinThreshold: config.MinThreshold,
		}
	}
	cl.config.UseCase.BuildRoute.FaultyPoolsConfig.SlippageConfigByGroup = slippageConfigByGroup
}

func (cl *ConfigLoader) setSafetyQuoteReduction(safetyQuoteConf valueobject.SafetyQuoteReductionConfig) {
	cl.config.UseCase.GetRoute.SafetyQuoteConfig.ExcludeOneSwapEnable = safetyQuoteConf.ExcludeOneSwapEnable
	cl.config.UseCase.GetRoute.SafetyQuoteConfig.Factor = safetyQuoteConf.Factor
	cl.config.UseCase.GetRoute.SafetyQuoteConfig.WhitelistedClient = safetyQuoteConf.WhitelistedClient
}

func (cl *ConfigLoader) setAlphaFeeConfig(alphaFeeConfig valueobject.AlphaFeeConfig) {
	cl.config.UseCase.GetRoute.AlphaFeeConfig = alphaFeeConfig
	cl.config.UseCase.BuildRoute.AlphaFeeConfig = alphaFeeConfig
	if duration, err := time.ParseDuration(alphaFeeConfig.TTL); err == nil {
		cl.config.Repository.AlphaFee.Redis.TTL = duration
	}
}

func (cl *ConfigLoader) setRFQAcceptableSlippageFraction(rfqAcceptableSlippageFraction int64) {
	cl.config.UseCase.BuildRoute.RFQAcceptableSlippageFraction = rfqAcceptableSlippageFraction
}

func (cl *ConfigLoader) setDexalotUpscalePercent(dexalotUpscalePercent int) {
	if rfqCfg, ok := cl.config.UseCase.BuildRoute.RFQ[dexalot.DexType]; ok {
		if dexalotCfg := rfqCfg.Properties; dexalotCfg != nil {
			dexalotCfg["upscale_percent"] = dexalotUpscalePercent
		}
	}
}

func (cl *ConfigLoader) setScaleHelperClients(scaleHelperClients []string) {
	cl.config.UseCase.GetRoute.ScaleHelperClients = scaleHelperClients
}

func (cl *ConfigLoader) setWhitelistedPrices(whitelistedPrices []string) {
	whitelistedPricesMap := lo.SliceToMap(whitelistedPrices, func(price string) (string, bool) {
		price = strings.ToLower(price)
		return price, true
	})

	cl.config.UseCase.GetRoute.AlphaFeeConfig.WhitelistPrices = whitelistedPricesMap
	cl.config.UseCase.GetRoute.AlphaFeeConfig.WhitelistPrices = whitelistedPricesMap
}
