package inverse

import "github.com/holiman/uint256"

// Extra is one atomic, idle-state snapshot. All integers are raw units (18
// decimals for ownership shares and WETH, 27 decimals for the rebase index).
// Value fields deliberately avoid aliasing across route-search branches.
type Extra struct {
	Version                uint8       `json:"version"`
	Live                   bool        `json:"live"`
	Inverse0               bool        `json:"inverse0"`
	FeeControllerSupported bool        `json:"feeControllerSupported"`
	ProtocolFee            uint32      `json:"protocolFee"`
	BlockNumber            uint64      `json:"blockNumber"`
	InitialShares          uint256.Int `json:"initialShares"`
	InitialQuote           uint256.Int `json:"initialQuote"`
	ReserveShares          uint256.Int `json:"reserveShares"`
	ReserveQuote           uint256.Int `json:"reserveQuote"`
	Index                  uint256.Int `json:"index"`
	CustodiedShares        uint256.Int `json:"custodiedShares"`
	NativeInverse          uint256.Int `json:"nativeInverse"`
	HookQuote              uint256.Int `json:"hookQuote"`
	NativeQuote            uint256.Int `json:"nativeQuote"`
	RoundingQuote          uint256.Int `json:"roundingQuote"`
	SqrtPriceX96           uint256.Int `json:"sqrtPriceX96"`
	Liquidity              uint256.Int `json:"liquidity"`
	Fees0                  uint256.Int `json:"fees0"`
	Fees1                  uint256.Int `json:"fees1"`
	Sequence               uint256.Int `json:"sequence"`
	Tick                   int         `json:"tick"`
}

type SwapInfo struct {
	Before Extra
	After  Extra
}
