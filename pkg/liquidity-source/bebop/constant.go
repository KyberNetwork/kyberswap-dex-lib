package bebop

import (
	"time"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

const (
	DexType = "bebop"

	MaxAge = time.Minute

	defaultGas = 200000
)

var (
	ErrLevelsTooOld          = errors.WithMessage(pool.ErrUnsupported, "levels too old")
	ErrEmptyPriceLevels      = errors.New("empty price levels")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)
