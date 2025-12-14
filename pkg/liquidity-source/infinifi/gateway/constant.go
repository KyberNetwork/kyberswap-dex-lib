package gateway

import (
	"errors"
)

const (
	DexType = "infinifi-gateway"

	// Gateway methods (from InfiniFiGatewayV2)
	gatewayMintAndStakeMethod = "mintAndStake" // USDC → siUSD (combined)
	gatewayMintAndLockMethod  = "mintAndLock"  // USDC → liUSD (combined)

	// ERC20 methods
	erc20BalanceOfMethod   = "balanceOf"
	erc20TotalSupplyMethod = "totalSupply"

	// ERC4626 methods (for siUSD StakedToken)
	erc4626ConvertToSharesMethod = "convertToShares" // Preview iUSD → siUSD conversion
	erc4626TotalAssetsMethod     = "totalAssets"     // Get total iUSD in siUSD vault

	// LockingController methods (for liUSD buckets)
	lockingControllerBucketsMethod = "buckets" // Get bucket data (shareToken, totalReceiptTokens, multiplier)

	// CoreControlled methods
	coreControlledPausedMethod = "paused"

	// Gas estimates - measured from actual transactions
	defaultMintAndStakeGas int64 = 650000 // USDC → siUSD (mint + stake combined)
	defaultMintAndLockGas  int64 = 650000 // USDC → liUSD (mint + lock combined)
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrContractPaused   = errors.New("contract paused")
	ErrSwapNotSupported = errors.New("swap path not supported")
	ErrAsyncRedemption  = errors.New("async redemptions not supported - use one-way paths only")
)
