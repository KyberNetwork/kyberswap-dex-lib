package config

import (
	"bytes"
	"context"
	"fmt"
	"log"
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
		cl.setEnabledDexes(remoteCfg.EnabledDexes)
		cl.setWhitelistedTokens(remoteCfg.WhitelistedTokens)
		cl.setBlacklistedPools(remoteCfg.BlacklistedPools)
		cl.setFeatureFlags(remoteCfg.FeatureFlags)
		cl.setLog(remoteCfg.Log)

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

func (cl *ConfigLoader) setEnabledDexes(enabledDexes []valueobject.Dex) {
	var stringDexes []string
	for _, d := range enabledDexes {
		stringDexes = append(stringDexes, string(d))
	}
	cl.config.EnableDexes = stringDexes
	cl.config.UseCase.GetRoutes.EnabledDexes = stringDexes
}

func (cl *ConfigLoader) setWhitelistedTokens(whitelistedTokens []valueobject.WhitelistedToken) {
	cl.config.WhitelistedTokens = whitelistedTokens
	cl.config.UseCase.GetRoutes.WhitelistedTokens = whitelistedTokens
}

func (cl *ConfigLoader) setBlacklistedPools(blacklistedPools []string) {
	cl.config.BlacklistedPools = blacklistedPools
	cl.config.UseCase.GetRoutes.BlacklistedPools = blacklistedPools
}

func (cl *ConfigLoader) setFeatureFlags(featureFlags valueobject.FeatureFlags) {
	cl.config.FeatureFlags = featureFlags
	cl.config.UseCase.GetRoutes.FeatureFlags = featureFlags
}

func (cl *ConfigLoader) setLog(log valueobject.Log) {
	cl.config.Log.Configuration.ConsoleLevel = log.ConsoleLevel
}
