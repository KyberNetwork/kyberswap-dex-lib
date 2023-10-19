package config

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"runtime"
	"strings"
	"sync"

	jsoniter "github.com/json-iterator/go"
	"github.com/mcuadros/go-defaults"
	"github.com/spf13/viper"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

const (
	EmptyConfigHash = ""
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type IRemoteConfigFetcher interface {
	Fetch(ctx context.Context) (valueobject.RemoteConfig, error)
}

type ConfigLoader struct {
	path                string
	config              *Config
	mu                  sync.RWMutex
	remoteConfigFetcher IRemoteConfigFetcher
}

// NewConfigLoader returns a new ConfigLoader.
func NewConfigLoader(path string) (*ConfigLoader, error) {
	cl := &ConfigLoader{path: path}
	err := cl.Initialize(context.Background())
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
		log.Println("Read config file failed. ", err)

		configBuffer, err := json.Marshal(c)

		if err != nil {
			return nil, err
		}

		err = viper.ReadConfig(bytes.NewBuffer(configBuffer))
		if err != nil {
			return nil, err
		}
	}

	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "__"))
	viper.AutomaticEnv()

	if err := viper.Unmarshal(c); err != nil {
		log.Printf("failed to unmarshal config %v\n", err)
		return nil, err
	}

	// set default gRPC config
	if c.GRPC.Host == "" && c.GRPC.Port == 0 {
		c.GRPC = ServerListen{
			Port: 10443,
			Host: "0.0.0.0",
		}
	}
	fmt.Println(viper.GetString("ENV"))
	fmt.Println("GOMAXPROCS: ", runtime.GOMAXPROCS(0))

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
func (cl *ConfigLoader) Initialize(_ context.Context) error {
	cfg, err := cl.GetLocalConfig()
	if err != nil {
		return err
	}

	// Set config
	cl.mu.Lock()
	cl.config = cfg
	cl.mu.Unlock()

	prettyJsonCfg, err := json.Marshal(cl.config)
	if err != nil {
		return err
	}

	fmt.Printf("local config: %+v\n", string(prettyJsonCfg))

	return nil
}

func (cl *ConfigLoader) Reload(ctx context.Context) error {
	remoteCfg, err := cl.GetRemoteConfig(ctx)
	if err != nil {
		fmt.Printf("failed to fetch config from remote: %s\n", err)
		return err
	}

	// Only override the config when remote config hash is NOT empty
	if remoteCfg.Hash != EmptyConfigHash {
		// Set config
		cl.mu.Lock()

		// Set each field so that it does not override the address of the pointer cl.config
		cl.setAvailableSources(remoteCfg.AvailableSources)
		cl.setWhitelistedTokens(remoteCfg.WhitelistedTokens)
		cl.setBlacklistedPools(remoteCfg.BlacklistedPools)
		cl.setFeatureFlags(remoteCfg.FeatureFlags)
		cl.setLog(remoteCfg.Log)
		cl.setFinderOptions(remoteCfg.FinderOptions)
		cl.setGetBestPoolOptions(remoteCfg.GetBestPoolsOptions)
		cl.setCacheConfig(remoteCfg.CacheConfig)
		cl.setBlacklistedRecipients(remoteCfg.BlacklistedRecipients)
		cl.setL2EncodePartners(remoteCfg.L2EncodePartners)
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
	cl.config.UseCase.GetRoute.AvailableSources = strAvailableSources
	cl.config.UseCase.GenerateBestPaths.AvailableSources = strAvailableSources
}

func (cl *ConfigLoader) setWhitelistedTokens(whitelistedTokens []valueobject.WhitelistedToken) {
	whitelistedTokenSet := make(map[string]bool, len(whitelistedTokens))
	for _, whitelistedToken := range whitelistedTokens {
		whitelistedTokenSet[strings.ToLower(whitelistedToken.Address)] = true
	}

	cl.config.Common.WhitelistedTokenSet = whitelistedTokenSet
	cl.config.UseCase.GetRoute.Aggregator.WhitelistedTokenSet = whitelistedTokenSet
	cl.config.UseCase.IndexPools.WhitelistedTokenSet = whitelistedTokenSet
	cl.config.UseCase.GenerateBestPaths.WhitelistedTokens = whitelistedTokens
}

func (cl *ConfigLoader) setBlacklistedPools(blacklistedPools []string) {
	blacklistedPoolSet := make(map[string]bool, len(blacklistedPools))
	for _, blacklistedPool := range blacklistedPools {
		blacklistedPoolSet[strings.ToLower(blacklistedPool)] = true
	}

	cl.config.Common.BlacklistedPoolsSet = blacklistedPoolSet
	cl.config.UseCase.PoolManager.BlacklistedPoolSet = blacklistedPoolSet
	cl.config.UseCase.GenerateBestPaths.BlacklistedPools = blacklistedPools
}

func (cl *ConfigLoader) setFeatureFlags(featureFlags valueobject.FeatureFlags) {
	cl.config.Common.FeatureFlags = featureFlags
	cl.config.UseCase.GetRoute.Aggregator.FeatureFlags = featureFlags
	cl.config.UseCase.BuildRoute.FeatureFlags = featureFlags
}

func (cl *ConfigLoader) setLog(log valueobject.Log) {
	cl.config.Log.Configuration.ConsoleLevel = log.ConsoleLevel
}

func (cl *ConfigLoader) setFinderOptions(finderOptions valueobject.FinderOptions) {
	cl.config.UseCase.GetRoute.Aggregator.FinderOptions = finderOptions
	cl.config.UseCase.GenerateBestPaths.SPFAFinderOptions = finderOptions
}

func (cl *ConfigLoader) setGetBestPoolOptions(getBestPoolsOptions valueobject.GetBestPoolsOptions) {
	cl.config.UseCase.GetRoute.Aggregator.GetBestPoolsOptions = getBestPoolsOptions
	cl.config.UseCase.GenerateBestPaths.GetBestPoolsOptions = getBestPoolsOptions
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

func (cl *ConfigLoader) setL2EncodePartners(l2EncodePartners []string) {
	l2EncodePartnersSet := make(map[string]struct{}, len(l2EncodePartners))
	for _, partner := range l2EncodePartners {
		l2EncodePartnersSet[strings.ToLower(partner)] = struct{}{}
	}

	cl.config.UseCase.BuildRoute.L2EncodePartners = l2EncodePartnersSet
}
