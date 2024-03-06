package generatepath

import (
	"time"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type (
	Config struct {
		GetBestPoolsOptions    valueobject.GetBestPoolsOptions `mapstructure:"getBestPoolsOptions"`
		GasTokenAddress        string                          `mapstructure:"gasTokenAddress"`
		BlacklistedPools       []string                        `mapstructure:"blacklistedPools"`
		WhitelistedTokens      []valueobject.WhitelistedToken  `mapstructure:"whitelistTokens"`
		SPFAFinderOptions      valueobject.FinderOptions       `mapstructure:"spfaFinderOptions"`
		AvailableSources       []string                        `mapstructure:"availableSources"`
		PathGeneratorDataTtl   time.Duration                   `mapstructure:"pathGeneratorDataTtl"`
		ConfigGeneratorDataTtl time.Duration                   `mapstructure:"configGeneratorDataTtl"`
		ChainID                valueobject.ChainID             `mapstructure:"chainId"`
	}

	TokenAndAmounts struct {
		TokenAddress string   `mapstructure:"tokenAddress"`
		Amounts      []string `mapstructure:"amounts"`
	}
)
