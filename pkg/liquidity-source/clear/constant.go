package clear

import "errors"

const (
	DexType = "clear"

	// Gas estimates
	defaultGas = 150000

	// ClearSwap contract methods
	methodPreviewSwap = "previewSwap"
	methodSwap        = "swap"

	// ClearFactory contract methods
	methodVaultsLength = "vaultsLength"
	methodVaults       = "vaults"

	// Default values
	defaultTokenDecimals = 18
	zeroString           = "0"
)

var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidAmountIn    = errors.New("invalid amount in")
	ErrInsufficientOutput = errors.New("insufficient output amount")
	ErrExtraEmpty         = errors.New("extra is empty")
	ErrStaticExtraEmpty   = errors.New("static extra is empty")
	ErrPoolNotFound       = errors.New("pool not found")
	ErrInvalidReserve     = errors.New("invalid reserve")
)
