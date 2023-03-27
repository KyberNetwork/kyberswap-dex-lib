package config

import (
	"fmt"

	"github.com/KyberNetwork/kyberswap-aggregator/pkg/logger"
)

type Common struct {
	ChainID    int      `mapstructure:"chainID"`
	DataFolder string   `mapstructure:"dataFolder"`
	PublicRPC  string   `mapstructure:"publicRPC"`
	RPCs       []string `mapstructure:"rpcs"`
	Address    Address  `mapstructure:"address"`
}
type EnableDexes []string
type ScanDex struct {
	Id         string                 `mapstructure:"id"`
	Handler    string                 `mapstructure:"handler"`
	Json       bool                   `mapstructure:"json"`
	Properties map[string]interface{} `mapstructure:"properties"`
}

type Gas struct {
	Default                int64  `mapstructure:"default" default:"125000"`
	GasPriceOracleContract string `mapstructure:"gasPriceOracleContract" json:"gasPriceOracleContract"`
}

type Address struct {
	Multicall       string `mapstructure:"multicall"`
	GasToken        string `mapstructure:"gasToken"`
	ExecutorAddress string `mapstructure:"executorAddress"`
	RouterAddress   string `mapstructure:"routerAddress"`
}

type CachePoint struct {
	Amount int `mapstructure:"amount"`
	TTL    int `mapstructure:"ttl"`
}

type CacheRange struct {
	FromUSD int `mapstructure:"fromUSD"`
	ToUSD   int `mapstructure:"toUSD"`
	TTL     int `mapstructure:"ttl"`
}

type LimitOrder struct {
	HTTPURL string `mapstructure:"httpUrl" default:""`
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

type TokenCatalog struct {
	HTTPURL string `mapstructure:"httpUrl" default:""`
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
