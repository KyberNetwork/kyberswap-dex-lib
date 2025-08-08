package helper

import (
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/decode"
	"github.com/ethereum/go-ethereum/common"
)

const customReceiverFlag = 0x01

func hasCustomReceiver(flags byte) bool {
	return flags&customReceiverFlag == customReceiverFlag
}

// DecodeSettlementPostInteractionData decodes SettlementPostInteractionData from bytes
// nolint: gomnd
// https://github.com/1inch/limit-order-sdk/blob/1793d32bd36c6cfea909caafbc15e8023a033249/src/limit-order/extensions/fee-taker/fee-taker.extension.ts#L104-L123
func DecodeSettlementPostInteractionData(data []byte) (SettlementPostInteractionData, error) {
	iter := decode.NewBytesIterator(data)
	if _, err := iter.NextUint160(); err != nil {
		return SettlementPostInteractionData{}, fmt.Errorf("skip address of extension: %w", err)
	}
	flags, err := iter.NextUint8()
	if err != nil {
		return SettlementPostInteractionData{}, fmt.Errorf(
			"get settlement post interaction data flags: %w", err)
	}

	integratorFeeRecipientBytes, err := iter.NextBytes(common.AddressLength)
	if err != nil {
		return SettlementPostInteractionData{}, fmt.Errorf("get integrator fee recipient: %w", err)
	}
	integratorFeeRecipient := common.BytesToAddress(integratorFeeRecipientBytes)

	protocolFeeRecipientBytes, err := iter.NextBytes(common.AddressLength)
	if err != nil {
		return SettlementPostInteractionData{}, fmt.Errorf("get protocol fee recipient: %w", err)
	}
	protocolFeeRecipient := common.BytesToAddress(protocolFeeRecipientBytes)

	var customReceiver *common.Address
	if hasCustomReceiver(flags) {
		customReceiverBytes, err := iter.NextBytes(common.AddressLength)
		if err != nil {
			return SettlementPostInteractionData{}, fmt.Errorf("get custom receiver: %w", err)
		}
		receiver := common.BytesToAddress(customReceiverBytes)
		customReceiver = &receiver
	}

	interactionData, err := ParseAmountData(iter)
	if err != nil {
		return SettlementPostInteractionData{}, fmt.Errorf("parse interaction data: %w", err)
	}

	var extraInteraction *Interaction
	if iter.HasMore() {
		interaction, err := DecodeInteraction(iter.RemainingData())
		if err != nil {
			return SettlementPostInteractionData{}, fmt.Errorf("decode extra interaction: %w", err)
		}
		extraInteraction = &interaction
	}

	return SettlementPostInteractionData{
		IntegratorFeeRecipient: integratorFeeRecipient,
		ProtocolFeeRecipient:   protocolFeeRecipient,
		CustomReceiver:         customReceiver,
		InteractionData:        interactionData,
		ExtraInteraction:       extraInteraction,
	}, nil
}
