package poolrank

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type PoolIndexRemovalParams struct {
	Pool                *entity.Pool
	Token0              string
	Token1              string
	IsToken0Whitelisted bool
	IsToken1Whitelisted bool
	TvlNative           float64
	AmplifiedTvlNative  float64
}
