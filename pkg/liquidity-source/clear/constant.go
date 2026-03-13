package clear

import "errors"

const (
	DexType = "clear"

	// Gas estimates
	defaultGas = 630965

	// ClearSwap contract methods
	methodTokens      = "tokens"
	methodTokenAssets = "tokenAssets"
	methodIouOf       = "iouOf"
	methodPreviewSwap = "previewSwap"

	// ClearFactory contract methods
	methodVaultsLength = "vaultsLength"
	methodVaults       = "vaults"
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrInvalidIOUToken  = errors.New("invalid iou token")
	ErrInvalidAmountIn  = errors.New("invalid amount in")
	ErrInvalidAmountOut = errors.New("invalid amount out")
	ErrPoolNotFound     = errors.New("pool not found")
)
