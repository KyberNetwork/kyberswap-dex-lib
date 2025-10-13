package arberazap

import "github.com/pkg/errors"

const (
	DexType        = "arbera-zap"
	defaultReserve = "100000000000000000000000000"
)

var (
	ErrInvalidToken      = errors.New("invalid token")
	ErrBasePoolsMismatch = errors.New("base pools mismatch")
)
