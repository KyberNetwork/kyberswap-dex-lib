package kyberpmm

import "errors"

var (
	ErrTokenNotFound          = errors.New("token not found")
	ErrNoPriceLevelsForPool   = errors.New("no price levels for pool")
	ErrEmptyPriceLevels       = errors.New("empty price levels")
	ErrInsufficientLiquidity  = errors.New("insufficient liquidity")
	ErrInvalidFirmQuoteParams = errors.New("invalid firm quote params")
	ErrNoSwapLimit            = errors.New("swap limit is required for PMM pools")
	ErrNotEnoughInventoryIn   = errors.New("not enough inventory in")
	ErrEmptyOrderList         = errors.New("order list from response is empty")
	ErrMakerAmountTooLow      = errors.New("actual maker amount is less than expected")
	ErrInvalidFeeAmount       = errors.New("invalid fee amount")
)
