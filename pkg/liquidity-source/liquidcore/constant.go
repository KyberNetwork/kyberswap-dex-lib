package liquidcore

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeLiquidCore

	defaultGas = 139441
)

var (
	ErrInvalidToken        = errors.New("invalid token")
	ErrZeroSwap            = errors.New("zero swap")
	ErrInsufficientReserve = errors.New("insufficient reserve")
)
