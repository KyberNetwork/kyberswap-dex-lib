package pool

import (
	"context"
	"math/big"
)

type RFQParams struct {
	// NetworkID blockchain network id
	NetworkID uint
	// Sender swap tx origin
	Sender string
	// Recipient fund recipient of swap tx
	Recipient string
	// RFQSender RFQ caller
	RFQSender string
	// RFQRecipient RFQ fund recipient
	RFQRecipient string
	// Slippage slippage tolerance (in bps) for RFQs that also aggregate dexes
	Slippage int64
	// SwapInfo swap info of the RFQ swap
	SwapInfo any

	Source string

	RequestID string

	AlphaFee string
}

type RFQResult struct {
	NewAmountOut *big.Int
	AlphaFee     *big.Int
	Extra        any
}

type RFQHandler struct{}

func (p *RFQHandler) RFQ(ctx context.Context, params RFQParams) (*RFQResult, error) {
	return nil, nil
}

func (p *RFQHandler) BatchRFQ(ctx context.Context, paramsSlice []RFQParams) ([]*RFQResult, error) {
	return nil, nil
}

func (p *RFQHandler) SupportBatch() bool {
	return false
}
