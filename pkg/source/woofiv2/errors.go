package woofiv2

import "errors"

var (
	ErrBaseTokenIsQuoteToken = errors.New("base token is quote token")
	ErrTokenInfoNotFound     = errors.New("token info is not found")
	ErrQuoteBalanceNotEnough = errors.New("quote balance is not enough")
	ErrBaseBalanceNotEnough  = errors.New("base balance is not enough")
	ErrBase2BalanceNotEnough = errors.New("base2 balance is not enough")
	ErrOracleNotFeasible     = errors.New("oracle is not feasible")
)
