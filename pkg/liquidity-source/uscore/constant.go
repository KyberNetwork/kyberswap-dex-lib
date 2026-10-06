package uscore

import (
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType             = valueobject.ExchangeUSCore
	defaultGas          = 122_000
	MaxAge              = 10 * time.Second
	multicall3          = "0xcA11bde05977b3631167028862bE2a173976CA11"
	maxRefinementRounds = 3
	maxSamplePoints     = 96
)

var ErrInvalidState = errors.New("uscore: invalid pool state")
