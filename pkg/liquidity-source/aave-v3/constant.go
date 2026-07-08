package aavev3

import "errors"

const (
	DexType = "aave-v3"

	supplyGas   int64 = 400000
	withdrawGas int64 = 350000

	poolMethodGetReservesList  = "getReservesList"
	poolMethodGetReserveData   = "getReserveData"
	poolMethodGetConfiguration = "getConfiguration"
)

var (
	ErrInvalidToken               = errors.New("invalid token")
	ErrSwapOutputExceedsLiquidity = errors.New("swap output exceeds available liquidity")
	ErrSwapInputExceedsSupplyCap  = errors.New("swap input exceeds supply cap")
)
