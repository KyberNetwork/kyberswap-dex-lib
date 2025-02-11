package primeeth

import (
	"errors"
)

const (
	DexType = "primeeth"

	primeZapper    = "0x3cf4db4c59dcb082d1a9719c54df3c04db93c6b7"
	lrtDepositPool = "0xa479582c8b64533102f6f528774c536e354b8d32"
	lrtConfig      = "0xf879c7859b6de6fadafb74224ff05b16871646bf"
	lrtOracle      = "0xA755c18CD2376ee238daA5Ce88AcF17Ea74C1c32"

	WETH     = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
	PrimeETH = "0x6ef3d766dfe02dc4bf04aae9122eb9a0ded25615"

	defaultReserves = "1000000000000000000000000"
)

const (
	lrtDepositPoolMethodPaused                = "paused"
	lrtDepositPoolMethodMinAmountToDeposit    = "minAmountToDeposit"
	lrtDepositPoolMethodGetTotalAssetDeposits = "getTotalAssetDeposits"

	lrtConfigMethodDepositLimitByAsset = "depositLimitByAsset"

	lrtOracleMethodPrimeETHPrice = "primeETHPrice"
)

var (
	defaultGas = Gas{
		Deposit: 250000,
	}
)

var (
	ErrPoolPaused                   = errors.New("pool is paused")
	ErrInvalidTokenIn               = errors.New("invalid tokenIn")
	ErrInvalidTokenOut              = errors.New("invalid tokenOut")
	ErrInvalidAmountToDeposit       = errors.New("invalid amount to deposit")
	ErrMaximumDepositLimitReached   = errors.New("maximum deposit limit reached")
	ErrMinimumAmountToReceiveNotMet = errors.New("minimum amount to receive not met")
)
