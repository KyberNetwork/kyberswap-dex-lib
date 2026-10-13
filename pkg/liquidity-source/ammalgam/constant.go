package ammalgam

import "errors"

const (
	DexType = "ammalgam"
	// Average of 30 cold fork measurements of tokenIn transfer + pair.swap on mainnet
	// block 26181046 (no daily lending checkpoint). Executor overhead is not included.
	defaultGas = int64(167_534)
)

var (
	ErrInvalidPool   = errors.New("invalid Ammalgam pool")
	ErrInvalidToken  = errors.New("invalid Ammalgam token")
	ErrInvalidAmount = errors.New("invalid Ammalgam amount")
	ErrZeroOutput    = errors.New("zero Ammalgam output")
	ErrReserveLimit  = errors.New("Ammalgam reserve exceeds uint112")
)
