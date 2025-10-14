package arberazap

import "github.com/pkg/errors"

const (
	DexType        = "arbera-zap"
	defaultReserve = "100000000000000000000000000"
)

var (
	ErrInvalidToken          = errors.New("invalid token")
	ErrBasePoolsMismatch     = errors.New("base pools mismatch")
	ErrBasePoolNotFound      = errors.New("base pool not found")
	ErrDenBuySellFeeNotFound = errors.New("den buy sell fee not found")
)
