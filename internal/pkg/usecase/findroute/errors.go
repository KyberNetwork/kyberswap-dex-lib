package findroute

import "errors"

var (
	ErrNoIPool          = errors.New("cannot get IPool from address")
	ErrNoPoolsFromToken = errors.New("no pool for fromToken")
	ErrNoInfoTokenIn    = errors.New("cannot get info for tokenIn")
	ErrNoInfoTokenOut   = errors.New("cannot get info for tokenOut")
)
