package config

import (
	"fmt"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/KyberNetwork/router-service/pkg/logger"
	"github.com/KyberNetwork/router-service/pkg/redis"
	"github.com/KyberNetwork/service-framework/pkg/client/grpcclient"
)

type Common struct {
	ChainID             valueobject.ChainID      `mapstructure:"chainId"`
	ChainName           string                   `mapstructure:"chainName"`
	RPC                 string                   `mapstructure:"rpc"`
	ExecutorAddress     string                   `mapstructure:"executorAddress"`
	RouterAddress       string                   `mapstructure:"routerAddress"`
	GasTokenAddress     string                   `mapstructure:"gasTokenAddress"`
	MulticallAddress    string                   `mapstructure:"multicallAddress"`
	WhitelistedTokenSet map[string]bool          `mapstructure:"whitelistedTokenSet"`
	BlacklistedPoolsSet map[string]bool          `mapstructure:"blacklistedPoolsSet"`
	AvailableSources    []string                 `mapstructure:"availableSources"`
	FeatureFlags        valueobject.FeatureFlags `mapstructure:"featureFlags"`
	SwaapAPIKey         string                   `mapstructure:"swaapAPIKey" json:"-"`
	HashflowAPIKey      string                   `mapstructure:"hashflowAPIKey" json:"-"`
}

type AEVM struct {
	// AEVM server URL
	AEVMServerURL string `mapstructure:"serverUrl"`

	// AddressesByDex Addresses needed to simulate a dex such as router and factory address.
	AddressesByDex map[string]map[string]string `mapstructure:"addressesByDex"`

	// Node URL for probing balance slot. The node must be enabled tracing feature.
	RPC string `mapstructure:"rpc"`

	// The wallet to probe balance slot for new tokens.
	FakeWallet string `mapstructure:"simulationWallet"`

	// Balance slots defined maunally
	PredefinedBalanceSlots map[string]*entity.ERC20BalanceSlot `mapstructure:"predefinedBalanceSlots"`

	// Use holders list (if available for token) as fallback if all faking balance strategies failed.
	UseHoldersListAsFallback bool `mapstructure:"useHoldersListAsFallback"`

	// The Redis storage where holders lists are maintained.
	TokenHoldersRedis redis.Config `mapstructure:"tokenHoldersRedis"`

	// Time-to-live of cached holders lists.
	CachedHoldersListTTLSec uint64 `mapstructure:"cachedHoldersListTTLSec"`
}

type Log struct {
	logger.Configuration `mapstructure:",squash"`
	SentryDSN            string `mapstructure:"sentryDSN" default:""`
}

type KeyPairInfo struct {
	StorageFilePath     string              `mapstructure:"storageFilePath" default:""`
	KeyIDForSealingData KeyIDForSealingData `mapstructure:"keyIDForSealingData" default:""`
}

type KeyIDForSealingData struct {
	ClientData string `mapstructure:"clientData" default:""`
}

// TODO: should move to grpc server when refactor builder
type ServerListen struct {
	Host string `yaml:"host" mapstructure:"host"`
	Port int    `yaml:"port" mapstructure:"port"`
}

// String return socket listen DSN
func (l ServerListen) String() string {
	return fmt.Sprintf("%s:%d", l.Host, l.Port)
}

type BlackjackConfig struct {
	GrpcConfig     grpcclient.Config `mapstructure:"grpcConfig"`
	CheckChunkSize int               `mapstructure:"checkChunkSize" default:"100"`
}
