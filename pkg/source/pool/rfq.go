package pool

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// RFQParams is the params for firm quote operations such as calling firm-quote API
type RFQParams struct {
	NetworkID    valueobject.ChainID // blockchain network id
	RequestID    string              // request id from getRoute
	Sender       string              // swap tx origin
	Recipient    string              // fund recipient of swap tx
	RFQSender    string              // RFQ caller (executor)
	RFQRecipient string              // RFQ fund recipient (executor/next pool/recipient)
	Source       string              // source client
	Slippage     int64               // tolerance (in bps) for RFQs that also aggregate dexes
	SwapInfo     any                 // swap info of the RFQ swap
	FeeInfo      any                 // generic fee info
}

// RFQResult is the result for firm quote operations
type RFQResult struct {
	NewAmountOut *big.Int
	Extra        any
}

// RFQHandler is the default no-op RFQ handler
type RFQHandler struct{}

func (p *RFQHandler) RFQ(_ context.Context, _ RFQParams) (*RFQResult, error) {
	return nil, nil
}

func (p *RFQHandler) BatchRFQ(_ context.Context, _ []RFQParams) ([]*RFQResult, error) {
	return nil, nil
}

func (p *RFQHandler) SupportBatch() bool {
	return false
}
