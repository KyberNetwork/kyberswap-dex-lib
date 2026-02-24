package gateway

import (
	"errors"
)

const (
	DexType = "infinifi-gateway"

	// ERC20 methods
	erc20TotalSupplyMethod = "totalSupply"

	// ERC4626 methods (for siUSD StakedToken)
	erc4626ConvertToSharesMethod = "convertToShares" // Preview iUSD → siUSD conversion
	erc4626TotalAssetsMethod     = "totalAssets"     // Get total iUSD in siUSD vault

	// LockingController methods (for liUSD buckets)
	lockingControllerBucketsMethod = "buckets" // Get bucket data (shareToken, totalReceiptTokens, multiplier)

	// CoreControlled methods
	coreControlledPausedMethod = "paused"
	defaultReserves            = "1000000000000000000000000"

	// Gas estimates
	defaultMintGas           int64 = 550000 // USDC → iUSD
	defaultStakeGas          int64 = 250000 // iUSD → siUSD
	defaultUnstakeGas        int64 = 250000 // siUSD → iUSD
	defaultRedeemGas         int64 = 250000 // iUSD → USDC
	defaultCreatePositionGas int64 = 250000 // iUSD → liUSD (lock)
	defaultMintAndStakeGas   int64 = 650000 // USDC → siUSD (combined)
	defaultMintAndLockGas    int64 = 650000 // USDC → liUSD (combined)
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrContractPaused   = errors.New("contract paused")
	ErrSwapNotSupported = errors.New("swap path not supported")
)
