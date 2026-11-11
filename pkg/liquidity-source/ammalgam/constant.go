package ammalgam

import "errors"

const (
	DexType = "ammalgam-dlex"
	// Average of 12 executeAmmalgam helper-frame gas measurements (6 mainnet pairs x 2 directions,
	// Tenderly sims). It includes validateOnUpdate's unthrottled lending checkpoint (~350k).
	defaultGas = int64(402_047)
)

var (
	ErrInvalidPool   = errors.New("invalid Ammalgam pool")
	ErrInvalidToken  = errors.New("invalid Ammalgam token")
	ErrInvalidAmount = errors.New("invalid Ammalgam amount")
	ErrZeroOutput    = errors.New("zero Ammalgam output")
	ErrReserveLimit  = errors.New("ammalgam reserve exceeds uint112")
)
