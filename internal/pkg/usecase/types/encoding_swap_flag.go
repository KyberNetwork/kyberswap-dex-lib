package types

type EncodingSwapFlag struct {
	Value uint32
}

var (
	// EncodingSwapFlagShouldNotKeepDustTokenOut is off when Executor contract should keep a wei of tokenOut.
	EncodingSwapFlagShouldNotKeepDustTokenOut EncodingSwapFlag

	// EncodingSwapFlagShouldApproveMax is on when Executor contract should approve max allowance for the pool.
	EncodingSwapFlagShouldApproveMax EncodingSwapFlag
)

func init() {
	EncodingSwapFlagShouldNotKeepDustTokenOut = EncodingSwapFlag{Value: 0x01}
	EncodingSwapFlagShouldApproveMax = EncodingSwapFlag{Value: 0x02}
}
