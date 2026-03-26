package canonic

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType          = valueobject.ExchangeCanonic
	defaultGas int64 = 625_092

	marketStateActive int64 = 0
)

var (
	feeDenom            = uint256.NewInt(1_000_000)
	rungDenom           = uint256.NewInt(100_000)
	priceSigfigs uint32 = 6
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidAmountIn       = errors.New("invalid amount in")
	ErrInvalidAmountOut      = errors.New("invalid amount out")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrMarketNotActive       = errors.New("market not active")
	ErrNoRungs               = errors.New("no rungs available")
	ErrZeroMidPrice          = errors.New("zero mid price")
	ErrInvalidState          = errors.New("invalid pool state")
)
