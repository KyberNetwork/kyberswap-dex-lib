package ringswapbacking

import "errors"

const (
	DexType = "ringswap-backing"

	feeNumerator   uint64 = 997
	feeDenominator uint64 = 1_000
)

var (
	ErrInvalidConfig       = errors.New("invalid ringswap-backing config")
	ErrInvalidToken        = errors.New("invalid token")
	ErrInvalidState        = errors.New("invalid ringswap-backing state")
	ErrInsufficientOutput  = errors.New("insufficient output")
	ErrInsufficientBacking = errors.New("insufficient deliverable backing")
	ErrNoSwapLimit         = errors.New("backing swap limit is required")
	ErrSourceAlreadyUsed   = errors.New("ringswap-backing source already used")
	ErrDuplicatePair       = errors.New("ringswap-backing pair configured more than once")
	ErrInvalidMetadata     = errors.New("invalid ringswap-backing metadata")
)
