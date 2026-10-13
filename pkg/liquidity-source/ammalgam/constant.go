package ammalgam

import "errors"

const (
	DexType    = "dlex"
	defaultGas = int64(180_000)
)

var (
	ErrInvalidPool   = errors.New("invalid Ammalgam pool")
	ErrInvalidToken  = errors.New("invalid Ammalgam token")
	ErrInvalidAmount = errors.New("invalid Ammalgam amount")
	ErrZeroOutput    = errors.New("zero Ammalgam output")
	ErrReserveLimit  = errors.New("Ammalgam reserve exceeds uint112")
)
