package uscoreprop

import (
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType    = valueobject.ExchangeUSCoreProp
	defaultGas = 122_000
	// MaxAge rejects ladders older than ~10 HyperEVM blocks (1s each): the pool prices off a live
	// oracle. pool-service must refresh well inside it (2s interval).
	MaxAge              = 10 * time.Second
	multicall3          = "0xcA11bde05977b3631167028862bE2a173976CA11"
	maxRefinementRounds = 3
	maxSamplePoints     = 96
)

var ErrInvalidState = errors.New("uscore-prop: invalid pool state")
