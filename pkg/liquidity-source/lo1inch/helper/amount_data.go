package helper

import (
	"fmt"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/bps"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/constants"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/decode"
	util "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/lo1inch/helper/utils"
)

type Fee struct {
	IntegratorFee     uint16 // In bps
	IntegratorShare   uint16 // In bps
	ResolverFee       uint16 // In bps
	WhitelistDiscount uint16 // In bps
}

type WhiteListDiscount struct {
	Discount  uint16 // In percent
	Addresses []util.AddressHalf
}

type AmountData struct {
	Fee               Fee
	WhiteListDiscount WhiteListDiscount
}

func ParseFee(iter *decode.BytesIterator) (Fee, error) {
	integratorFee, err := iter.NextUint16()
	if err != nil {
		return Fee{}, fmt.Errorf("get intergator fee: %w", err)
	}

	integratorShare, err := iter.NextUint8()
	if err != nil {
		return Fee{}, fmt.Errorf("get intergator share: %w", err)
	}

	resolverFee, err := iter.NextUint16()
	if err != nil {
		return Fee{}, fmt.Errorf("get resolver fee: %w", err)
	}

	whitelistDiscountSub, err := iter.NextUint8()
	if err != nil {
		return Fee{}, fmt.Errorf("get whitelist discount: %w", err)
	}

	whitelistDiscount := int(constants.FeeBase1e2.Int64()) - int(whitelistDiscountSub)
	return Fee{
		IntegratorFee:     bps.FromFraction(int(integratorFee), constants.FeeBase1e5),
		IntegratorShare:   bps.FromFraction(int(integratorShare), constants.FeeBase1e2),
		ResolverFee:       bps.FromFraction(int(resolverFee), constants.FeeBase1e5),
		WhitelistDiscount: bps.FromFraction(whitelistDiscount, constants.FeeBase1e2),
	}, nil
}

func ParseAmountData(iter *decode.BytesIterator) (AmountData, error) {
	fee, err := ParseFee(iter)
	if err != nil {
		return AmountData{}, fmt.Errorf("parse fee: %w", err)
	}

	whitelistFromAmountSize, err := iter.NextUint8()
	if err != nil {
		return AmountData{}, fmt.Errorf("get whitelist from amount size: %w", err)
	}
	addresses := make([]util.AddressHalf, whitelistFromAmountSize)
	for i := 0; i < int(whitelistFromAmountSize); i++ {
		addressHalfBytes, err := iter.NextBytes(util.AddressHalfLength)
		if err != nil {
			return AmountData{}, fmt.Errorf("get whitelist item address half: %w", err)
		}
		addresses[i] = util.BytesToAddressHalf(addressHalfBytes)
	}

	return AmountData{
		Fee: fee,
		WhiteListDiscount: WhiteListDiscount{
			Discount:  fee.WhitelistDiscount,
			Addresses: addresses,
		},
	}, nil
}
