package types

type EncodingFlag struct {
	Name  string
	Value int64
}

var (
	// EncodingFlagPartialFill (not used yet) is true when we enable partial fill
	EncodingFlagPartialFill EncodingFlag

	// EncodingFlagRequireExtraEth is on when tokenIn is eth and amount in > total swap amount (ex: charge fee by currency_in)
	EncodingFlagRequireExtraEth EncodingFlag

	// EncodingFlagShouldClaim (not used yet) is on when router should collect token from msg.sender, only used in swapGeneric (meta aggregator)
	EncodingFlagShouldClaim EncodingFlag

	// EncodingFlagBurnFromMsgSender (not used yet) no usage in contract
	EncodingFlagBurnFromMsgSender EncodingFlag

	// EncodingFlagBurnFromTxOrigin (not used yet) no usage in contract
	EncodingFlagBurnFromTxOrigin EncodingFlag

	// EncodingFlagSimpleSwap is on when swapping in simple mode
	EncodingFlagSimpleSwap EncodingFlag

	// EncodingFlagFeeOnDst is on when there is fee charged by currency_out
	EncodingFlagFeeOnDst EncodingFlag

	// EncodingFlagFeeInBps is on when fee amount is in bps
	EncodingFlagFeeInBps EncodingFlag

	// EncodingFlagApproveFund (not used yet) approve allowance to `SwapExecutionParams.approveTarget`
	EncodingFlagApproveFund EncodingFlag
)

func init() {
	EncodingFlagPartialFill = EncodingFlag{Name: "_PARTIAL_FILL", Value: 0x01}
	EncodingFlagRequireExtraEth = EncodingFlag{Name: "_REQUIRES_EXTRA_ETH", Value: 0x02}
	EncodingFlagShouldClaim = EncodingFlag{Name: "_SHOULD_CLAIM", Value: 0x04}
	EncodingFlagBurnFromMsgSender = EncodingFlag{Name: "_BURN_FROM_MSG_SENDER", Value: 0x08}
	EncodingFlagBurnFromTxOrigin = EncodingFlag{Name: "_BURN_FROM_TX_ORIGIN", Value: 0x10}
	EncodingFlagSimpleSwap = EncodingFlag{Name: "_SIMPLE_SWAP", Value: 0x20}
	EncodingFlagFeeOnDst = EncodingFlag{Name: "_FEE_ON_DST", Value: 0x40}
	EncodingFlagFeeInBps = EncodingFlag{Name: "_FEE_IN_BPS", Value: 0x80}
	EncodingFlagApproveFund = EncodingFlag{Name: "_APPROVE_FUND", Value: 0x100}
}
