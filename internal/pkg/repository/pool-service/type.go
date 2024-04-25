package poolservice

import (
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type Config struct {
	BaseURL  string              `json:"baseUrl" mapstructure:"baseUrl"`
	Timeout  time.Duration       `json:"timeout" mapstructure:"timeout"`
	Insecure bool                `json:"insecure" mapstructure:"insecure"`
	ClientID string              `json:"clientId" mapstructure:"clientId"`
	ChainID  valueobject.ChainID `json:"chainId" mapstructure:"chainId"`
}
