package ambient

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/pkg/errors"
)

const (
	DexType        = "ambient"
	DexTypeAmbient = DexType

	defaultSubgraphLimit = 1000
)

var (
	NativeTokenPlaceholderAddress = common.Address{}
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrPairNotFound      = errors.New("pair not found")
	ErrNoTrackedPairs    = errors.New("no tracked pairs")
	ErrZeroAmount        = errors.New("zero amount")
	ErrInsufficientFund  = errors.New("insufficient reserve")
	ErrTickRangeExceeded = errors.New("swap exceeds fetched tick range")
)
