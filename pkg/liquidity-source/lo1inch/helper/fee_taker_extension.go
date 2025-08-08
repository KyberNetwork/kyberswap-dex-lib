package helper

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/decode"
	util "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/utils"
	"github.com/ethereum/go-ethereum/common"
)

type FeeTakerExtension struct {
	Address          common.Address
	Fees             Fees
	Whitelist        Whitelist
	MakerPermit      *Interaction
	ExtraInteraction *Interaction
	CustomReceiver   *common.Address
	feeCalculator    FeeCalculator
}

var ErrInvalidExtension = errors.New("invalid extension")

// https://github.com/1inch/limit-order-sdk/blob/1793d32bd36c6cfea909caafbc15e8023a033249/src/limit-order/extensions/fee-taker/fee-taker.extension.ts#L84
//
//nolint:funlen,cyclop
func NewFeeTakerFromExtension(extension Extension) (FeeTakerExtension, error) {
	if len(extension.MakingAmountData) == 0 {
		return FeeTakerExtension{},
			fmt.Errorf("%w: making amount data is empty", ErrInvalidExtension)
	}

	extensionAddress := util.AddressFromFirstBytes(extension.MakingAmountData)
	if len(extension.TakingAmountData) == 0 || util.AddressFromFirstBytes(extension.TakingAmountData) != extensionAddress {
		return FeeTakerExtension{},
			fmt.Errorf("%w: taking amount data settlement contract mismatch", ErrInvalidExtension)
	}
	if len(extension.PostInteraction) == 0 || util.AddressFromFirstBytes(extension.PostInteraction) != extensionAddress {
		return FeeTakerExtension{},
			fmt.Errorf("%w: post interaction settlement contract mismatch", ErrInvalidExtension)
	}
	if !bytes.Equal(extension.TakingAmountData, extension.MakingAmountData) {
		return FeeTakerExtension{},
			fmt.Errorf("%w: takingAmountData and makingAmountData not match", ErrInvalidExtension)
	}

	postInteractionData, err := DecodeSettlementPostInteractionData(extension.PostInteraction)
	if err != nil {
		return FeeTakerExtension{}, fmt.Errorf("decode post interaction data: %w", err)
	}

	amountIter := decode.NewBytesIterator(extension.MakingAmountData)
	if _, err := amountIter.NextUint160(); err != nil {
		return FeeTakerExtension{}, fmt.Errorf("skip address of extension: %w", err)
	}
	amountData, err := ParseAmountData(amountIter)
	if err != nil {
		return FeeTakerExtension{}, fmt.Errorf("decode amount data: %w", err)
	}

	var permit *Interaction
	if extension.HasMakerPermit() {
		permitInteraction, err := DecodeInteraction(extension.MakerPermit)
		if err != nil {
			return FeeTakerExtension{}, fmt.Errorf("decode maker permit: %w", err)
		}
		permit = &permitInteraction
	}

	if amountData.Fee.IntegratorFee != postInteractionData.InteractionData.Fee.IntegratorFee {
		return FeeTakerExtension{}, fmt.Errorf("%w: integrator fee not match", ErrInvalidExtension)
	}
	if amountData.Fee.ResolverFee != postInteractionData.InteractionData.Fee.ResolverFee {
		return FeeTakerExtension{}, fmt.Errorf("%w: resolver fee", ErrInvalidExtension)
	}
	if amountData.Fee.WhitelistDiscount != postInteractionData.InteractionData.Fee.WhitelistDiscount {
		return FeeTakerExtension{}, fmt.Errorf("%w: whitelist discount not match", ErrInvalidExtension)
	}
	if amountData.Fee.IntegratorShare != postInteractionData.InteractionData.Fee.IntegratorShare {
		return FeeTakerExtension{}, fmt.Errorf("%w: integrator share not match", ErrInvalidExtension)
	}
	for i, item := range postInteractionData.InteractionData.WhiteListDiscount.Addresses {
		if item != amountData.WhiteListDiscount.Addresses[i] {
			return FeeTakerExtension{}, fmt.Errorf("%w: whitelist address not match", ErrInvalidExtension)
		}
	}
	feeTakerExtension := FeeTakerExtension{
		Address: extensionAddress,
		Whitelist: Whitelist{
			Addresses: amountData.WhiteListDiscount.Addresses,
		},
		MakerPermit:      permit,
		ExtraInteraction: postInteractionData.ExtraInteraction,
		CustomReceiver:   postInteractionData.CustomReceiver,
	}
	if amountData.Fee.ResolverFee != 0 {
		feeTakerExtension.Fees.Resolver = ResolverFee{
			Receiver:          postInteractionData.ProtocolFeeRecipient,
			Fee:               amountData.Fee.ResolverFee,
			WhitelistDiscount: amountData.WhiteListDiscount.Discount,
		}
	}
	if amountData.Fee.IntegratorFee != 0 {
		feeTakerExtension.Fees.Integrator = IntegratorFee{
			Integrator: postInteractionData.IntegratorFeeRecipient,
			Protocol:   postInteractionData.ProtocolFeeRecipient,
			Fee:        amountData.Fee.IntegratorFee,
			Share:      amountData.Fee.IntegratorShare,
		}
	}

	feeTakerExtension.feeCalculator = FeeCalculator{
		fees:      feeTakerExtension.Fees,
		whitelist: feeTakerExtension.Whitelist,
	}
	return feeTakerExtension, nil
}

// GetTakingAmount return taking amount with fee applied
func (f *FeeTakerExtension) GetTakingAmount(taker common.Address, takingAmount *big.Int) *big.Int {
	return f.feeCalculator.GetTakingAmount(taker, takingAmount)
}

// GetMakingAmount return making amount with fee applied
func (f *FeeTakerExtension) GetMakingAmount(taker common.Address, makingAmount *big.Int) *big.Int {
	return f.feeCalculator.GetMakingAmount(taker, makingAmount)
}

// GetResolverFee which resolver pays to resolver fee receiver
func (f *FeeTakerExtension) GetResolverFee(taker common.Address, takingAmount *big.Int) *big.Int {
	return f.feeCalculator.GetResolverFee(taker, takingAmount)
}

// GetIntegratorFee which integrator gets to integrator wallet
func (f *FeeTakerExtension) GetIntegratorFee(taker common.Address, takingAmount *big.Int) *big.Int {
	return f.feeCalculator.GetIntegratorFee(taker, takingAmount)
}

// GetProtocolShareOfIntegratorFee which protocol gets as share from integrator fee
func (f *FeeTakerExtension) GetProtocolShareOfIntegratorFee(taker common.Address, takingAmount *big.Int) *big.Int {
	return f.feeCalculator.GetProtocolShareOfIntegratorFee(taker, takingAmount)
}

// GetProtocolFee which protocol gets as share from resolver fee
func (f *FeeTakerExtension) GetProtocolFee(taker common.Address, takingAmount *big.Int) *big.Int {
	return f.feeCalculator.GetProtocolFee(taker, takingAmount)
}
