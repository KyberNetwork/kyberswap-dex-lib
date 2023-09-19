package router

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
)

type SwapExecutionParams struct {
	// CallTarget address of aggregation executor
	CallTarget common.Address `json:"callAddress"`

	// ApproveTarget only use for other aggregator than KyberSwap Aggregator
	ApproveTarget common.Address `json:"approveTarget"`

	// TargetData packed data used by aggregation executor
	TargetData []byte `json:"targetData"`

	// Desc contains data used by aggregation router
	Desc SwapDescriptionV2 `json:"desc"`

	// ClientData data to be emitted by ClientData event
	ClientData []byte `json:"clientData"`
}

type SwapSimpleModeInputs struct {
	// Caller address of aggregation executor
	Caller common.Address `json:"caller"`

	// Desc contains data used by aggregation router
	Desc SwapDescriptionV2 `json:"desc"`

	// ExecutorData packed data used by aggregation executor
	ExecutorData []byte `json:"executorData"`

	// ClientData data to be emitted by ClientData event
	ClientData []byte `json:"clientData"`
}

type SwapDescriptionV2 struct {
	// SrcToken address of token to be swapped
	SrcToken common.Address `json:"srcToken"`

	// DstToken address of token to be received
	DstToken common.Address `json:"dstToken"`

	// SrcReceivers addresses aggregation router should transfer token to
	SrcReceivers []common.Address `json:"srcReceivers"`

	// SrcAmounts amounts aggregation router should transfer token
	SrcAmounts []*big.Int `json:"srcAmounts"`

	// FeeReceivers address to be received fee
	FeeReceivers []common.Address `json:"feeReceivers"`

	// FeeAmounts amounts of fee to be received
	FeeAmounts []*big.Int `json:"feeAmounts"`

	// DstReceiver address to be received DstToken
	DstReceiver common.Address `json:"dstReceiver"`

	// Amount of SrcToken (before fee)
	Amount *big.Int `json:"amount"`

	// MinReturnAmount minimum amount of DstToken to be received
	MinReturnAmount *big.Int `json:"minReturnAmount"`

	// Flags for aggregation router contract operations
	Flags uint32 `json:"flags"`

	// Permit contains signed approval
	Permit []byte `json:"permit"`
}

type SwapDataCompress struct {
	Data []byte `json:"data"`
}
