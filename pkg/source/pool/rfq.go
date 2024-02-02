package pool

import "math/big"

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
	// SwapInfo swap info of the RFQ swap
	SwapInfo any
}

type RFQResult struct {
	NewAmountOut *big.Int
	Extra        any
}
