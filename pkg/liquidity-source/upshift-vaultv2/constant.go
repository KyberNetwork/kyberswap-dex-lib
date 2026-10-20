package upshiftvaultv2

import "errors"

const (
	DexType = "upshift-vaultv2"

	// Fork gasUsed - 21000 of deposit() to a fresh receiver, exact integer average of 10 (earnAUSD)
	// and 15 (HLPe) deposits. defaultGas is the average of the two, for vaults not measured.
	gasEarnAUSD int64 = 132956
	gasHLPe     int64 = 106578
	defaultGas  int64 = 119767

	earnAUSDVault = "0x36edbf0c834591bfdfcac0ef9605528c75c406aa"
	hlpeVault     = "0x8ffdcd8a96d293f45aa044d10b899f9d71897e8a"

	vaultMethodAsset          = "asset"
	vaultMethodLP             = "lpTokenAddress"
	vaultMethodAssetsWL       = "assetsWhitelistAddress"
	vaultMethodSendersWL      = "sendersWhitelistAddress"
	vaultMethodDepositsPaused = "depositsPaused"
	vaultMethodExternalAssets = "externalAssets"
	vaultMethodMaxDeposit     = "maxDepositAmount"
	vaultMethodTotalAssets    = "getTotalAssets"
	vaultMethodDepositCap     = "depositCap"
	wlMethodValuation         = "getTotalAssetsValuation"
	erc20MethodTotalSupply    = "totalSupply"
)

var (
	ErrInvalidSwapPath = errors.New("upshift-vaultv2: only reference asset -> LP token is supported")
	ErrInvalidExtra    = errors.New("upshift-vaultv2: invalid pool extra")
	ErrZeroAmount      = errors.New("upshift-vaultv2: zero amount")
	ErrOverflow        = errors.New("upshift-vaultv2: uint256 arithmetic overflow")
	ErrDepositsPaused  = errors.New("upshift-vaultv2: deposits paused")
	ErrSenderGated     = errors.New("upshift-vaultv2: senders whitelist is set")
	ErrNoSupply        = errors.New("upshift-vaultv2: empty vault, share formula unknown")
	ErrInsufficientShr = errors.New("upshift-vaultv2: zero shares")
	ErrMaxDeposit      = errors.New("upshift-vaultv2: max deposit amount reached")
	ErrDepositCap      = errors.New("upshift-vaultv2: deposit cap reached")
)
