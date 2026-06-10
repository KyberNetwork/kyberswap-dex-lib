package gateway

import (
	"errors"
)

const (
	DexType = "infinifi-gateway"

	// ERC20 methods
	erc20TotalSupplyMethod = "totalSupply"

	// ERC4626 methods (for siUSD StakedToken)
	erc4626TotalAssetsMethod = "totalAssets" // Get total iUSD in siUSD vault

	// LockingController methods (for liUSD buckets)
	lockingControllerBucketsMethod = "buckets" // Get bucket data (shareToken, totalReceiptTokens, multiplier)

	// CoreControlled methods
	coreControlledPausedMethod = "paused"
	defaultReserves            = "100000000000000"

	// Gas estimates
	gasMint             int64 = 1002480 // USDC → iUSD
	gasStake            int64 = 171489  // iUSD → siUSD
	gasUnstake          int64 = 2443171 // siUSD → iUSD
	gasRedeem           int64 = 3608595 // iUSD → USDC
	gasCreatePosition   int64 = 155769  // iUSD → liUSD (lock)
	gasMintAndStake     int64 = 1147226 // USDC → siUSD (combined)
	gasMintAndLock      int64 = 1114675 // USDC → liUSD (combined)
	gasUnstakeAndRedeem int64 = 4431817 // siUSD → USDC (combined)
)

var (
	ErrContractPaused   = errors.New("contract paused")
	ErrSwapNotSupported = errors.New("swap path not supported")
)
