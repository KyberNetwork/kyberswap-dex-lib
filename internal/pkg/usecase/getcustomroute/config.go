package getcustomroute

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/getroute"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type Config struct {
	ChainID                 valueobject.ChainID `mapstructure:"chainId" json:"chainId"`
	RouterAddress           string              `mapstructure:"routerAddress" json:"routerAddress"`
	GasTokenAddress         string              `mapstructure:"gasTokenAddress" json:"gasTokenAddress"`
	AvailableSources        []string            `mapstructure:"availableSources" json:"availableSources"`
	UnscalableSources       []string            `mapstructure:"unscalableSources" json:"unscalableSources"`
	ExcludedSourcesByClient map[string][]string `mapstructure:"excludedSourcesByClient" json:"excludedSourcesByClient"`

	Aggregator getroute.AggregatorConfig `mapstructure:"aggregator" json:"aggregator"`
}
