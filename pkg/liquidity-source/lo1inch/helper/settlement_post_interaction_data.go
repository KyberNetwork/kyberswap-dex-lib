package helper

import (
	"github.com/ethereum/go-ethereum/common"
)

type SettlementPostInteractionData struct {
	IntegratorFeeRecipient common.Address
	ProtocolFeeRecipient   common.Address
	CustomReceiver         *common.Address
	InteractionData        AmountData
	ExtraInteraction       *Interaction
}

func (s SettlementPostInteractionData) HasFees() bool {
	return s.IntegratorFeeRecipient != (common.Address{}) || s.ProtocolFeeRecipient != (common.Address{})
}
