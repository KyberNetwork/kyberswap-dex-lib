package pool

import (
	"context"
	"math/big"
)

type RFQParams struct {
	NetworkID    uint   // blockchain network id
	Sender       string // swap tx origin
	Recipient    string // fund recipient of swap tx
	RFQSender    string // RFQ caller
	RFQRecipient string // RFQ fund recipient
	Slippage     int64  // tolerance (in bps) for RFQs that also aggregate dexes
	SwapInfo     any    // swap info of the RFQ swap
	Source       string // source client
}

type RFQResult struct {
	NewAmountOut *big.Int
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
