package printr

import "errors"

const (
	DexType = "printr"

	printrMethodGetCurve   = "getCurve"
	printrMethodTradingFee = "tradingFee"
	printrMethodPaused     = "paused"
)

const (
	buyGas  = 150000
	sellGas = 120000
)

var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrTokenGraduated       = errors.New("token graduated")
	ErrContractPaused       = errors.New("contract paused")
	ErrInsufficientReserves = errors.New("insufficient reserves")
	ErrZeroAmount           = errors.New("zero amount")
)
