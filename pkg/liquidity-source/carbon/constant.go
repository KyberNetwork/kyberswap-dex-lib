package carbon

import (
	"errors"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeCarbon

	orderIdxToken0 = 0
	orderIdxToken1 = 1

	one           = 1 << 48
	ppmResolution = 1000000

	defaultTradingFeePpm = 4000

	defaultSingleTradeActionGas   = 11763
	defaultTradeBySourceAmountGas = 63037

	maxStrategiesPerBatch = 200
)

var (
	uOne           = uint256.NewInt(one)
	oneSquared     = new(uint256.Int).Mul(uOne, uOne)
	uPmmResolution = uint256.NewInt(ppmResolution)

	ErrInvalidToken          = errors.New("invalid token")
	ErrInvalidSwap           = errors.New("invalid swap")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrNoStrategies          = errors.New("no strategies available")
	ErrZeroAmount            = errors.New("zero amount")
)
