package gateway

import (
	"errors"
)

const (
	DexType = "infinifi-gateway"

	// Gateway methods (from InfiniFiGatewayV2)
	// Only synchronous operations are supported
	gatewayMintMethod           = "mint"           // USDC → iUSD (synchronous)
	gatewayStakeMethod          = "stake"          // iUSD → siUSD (synchronous)
	gatewayCreatePositionMethod = "createPosition" // iUSD → liUSD (synchronous)

	// ERC20 methods
	erc20BalanceOfMethod   = "balanceOf"
	erc20TotalSupplyMethod = "totalSupply"

	// ERC4626 methods (for siUSD StakedToken)
	erc4626ConvertToSharesMethod = "convertToShares" // Preview iUSD → siUSD conversion
	erc4626TotalAssetsMethod     = "totalAssets"     // Get total iUSD in siUSD vault

	// CoreControlled methods
	coreControlledPausedMethod = "paused"

	// Gas estimates - measured from actual transactions
	// Note: These are for synchronous operations only
	defaultMintGas  int64 = 150000
	defaultStakeGas int64 = 200000 // Higher due to YieldSharing.distributeInterpolationRewards()
	defaultLockGas  int64 = 250000 // For createPosition (locking to liUSD)
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrContractPaused   = errors.New("contract paused")
	ErrSwapNotSupported = errors.New("swap path not supported")
	ErrAsyncRedemption  = errors.New("async redemptions not supported - use one-way paths only")
)
