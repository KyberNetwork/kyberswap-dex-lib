package helper

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type SettlementPostInteractionData struct {
	IntegratorFeeRecipient common.Address
	ProtocolFeeRecipient   common.Address
	CustomReceiver         *common.Address
	InteractionData        AmountData
	ExtraInteraction       *Interaction
}

func (s SettlementPostInteractionData) HasFees() bool {
	return s.IntegratorFeeRecipient != valueobject.AddrZero || s.ProtocolFeeRecipient != valueobject.AddrZero
}
