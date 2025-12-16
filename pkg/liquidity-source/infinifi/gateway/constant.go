package gateway

import (
	"errors"
)

const (
	DexType = "infinifi-gateway"

	// Gateway methods (from InfiniFiGatewayV2)
	gatewayMintMethod           = "mint"           // USDC → iUSD
	gatewayStakeMethod          = "stake"          // iUSD → siUSD
	gatewayUnstakeMethod        = "unstake"        // siUSD → iUSD
	gatewayRedeemMethod         = "redeem"         // iUSD → USDC
	gatewayCreatePositionMethod = "createPosition" // iUSD → liUSD

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
	defaultMintGas           int64 = 150000 // USDC → iUSD
	defaultStakeGas          int64 = 250000 // iUSD → siUSD
	defaultUnstakeGas        int64 = 250000 // siUSD → iUSD
	defaultRedeemGas         int64 = 200000 // iUSD → USDC
	defaultCreatePositionGas int64 = 200000 // iUSD → liUSD (lock)
)

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrContractPaused   = errors.New("contract paused")
	ErrSwapNotSupported = errors.New("swap path not supported")
)
