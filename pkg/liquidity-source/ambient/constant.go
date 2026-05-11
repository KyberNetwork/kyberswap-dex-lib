package ambient

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeAmbient

	indexerPoolListPath = "/gcgo/pool_list"
)

var defaultGas = Gas{
	BaseGas:          120000,
	CrossInitTickGas: 21000,
	PinSpillGas:      12681,
	KnockoutCrossGas: 31093,
}

var (
	ErrPairNotFound      = errors.New("pair not found")
	ErrNoTrackedPairs    = errors.New("no tracked pairs")
	ErrZeroAmount        = errors.New("zero amount")
	ErrInsufficientFund  = errors.New("insufficient reserve")
	ErrTickRangeExceeded = errors.New("swap exceeds fetched tick range")
)
