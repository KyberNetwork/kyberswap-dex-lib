package stake

import "github.com/holiman/uint256"

// StaticExtra is immutable per pool instance -- set once at listing time, never refreshed by the
// tracker (unlike Extra).
type StaticExtra struct {
	// IsNativeUnderlying reports whether this instance's underlying token was configured as
	// valueobject.NativeAddress, i.e. its deposit()/instantWithdraw() genuinely move native
	// currency rather than pulling/pushing an ERC20. Governs both SwapReceiveNativeIn (deposit
	// leg) and SwapReturnNativeOut (withdraw leg), since on the one real contract today both
	// directions move native together.
	IsNativeUnderlying bool `json:"isNativeUnderlying"`
}

type Extra struct {
	Paused bool `json:"paused"`

	// DepositRate = convertBnbToSnBnb(1e18): slisBNB minted per 1e18 wei of WBNB deposited.
	DepositRate *uint256.Int `json:"depositRate"`
	// WithdrawRate = convertSnBnbToBnb(1e18): BNB returned per 1e18 wei of slisBNB burned.
	WithdrawRate *uint256.Int `json:"withdrawRate"`

	// InstantWithdrawFeeRate is TEN_DECIMALS(1e10)-scaled, applied only on the withdraw leg.
	InstantWithdrawFeeRate *uint256.Int `json:"instantWithdrawFeeRate"`
	// MinBnb is the minimum BNB output instantWithdraw allows.
	MinBnb *uint256.Int `json:"minBnb"`

	// InstantWithdrawEligible mirrors instantWhitelistOff(): the withdraw leg is only ever
	// quoted when Lista has opened instant withdraw to everyone.
	InstantWithdrawEligible bool `json:"instantWithdrawEligible"`
}

// Meta is returned from GetMetaInfo(tokenIn, tokenOut) and becomes swap.PoolExtra downstream in
// aggregator-encoding -- the channel other dual-direction pools (e.g. lista/stable's own
// TokenInIsNative/TokenOutIsNative) already use to hand the encoder direction-specific facts.
type Meta struct {
	BlockNumber uint64 `json:"blockNumber"`
	// IsWithdraw is true when tokenIn is the share token, i.e. this leg is instantWithdraw
	// rather than deposit.
	IsWithdraw bool `json:"isWithdraw"`
	// IsNativeUnderlying mirrors StaticExtra.IsNativeUnderlying -- the encoder's wrap/unwrap
	// decision (aggregator-encoding's swapReceiveNativeIn/swapReturnNativeOut) reads this instead
	// of inferring native support from whether the token address happens to be wrapped-native.
	IsNativeUnderlying bool `json:"isNativeUnderlying"`
}
