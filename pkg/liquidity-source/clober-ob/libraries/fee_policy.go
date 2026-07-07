package cloberlib

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
	"github.com/samber/lo"

	u256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type FeePolicy uint32

var (
	ratePrecision  = u256.TenPow(6)
	maxFeeRateU256 = uint256.NewInt(500000)

	rateMask = u256.New("8388607") // 0x7fffff 23 bits
)

func (fp FeePolicy) UsesQuote() bool {
	return (fp>>23)&1 == 1
}

func (fp FeePolicy) Rate() int32 {
	res := uint256.NewInt(uint64(fp))
	res.And(res, rateMask)
	res.Sub(res, maxFeeRateU256)

	return int32(res.Uint64())
}

func (fp FeePolicy) CalculateFee(amount *uint256.Int, reverseRounding bool) *int256.Int {
	r := fp.Rate()

	positive := r > 0
	absRate := uint256.NewInt(uint64(r))

	absFee := new(uint256.Int)
	u256.MulDivRounding(absFee, amount, absRate, ratePrecision, lo.Ternary(reverseRounding, !positive, positive))

	return lo.Ternary(positive, u256.SInt256, u256.SNeg)(absFee)
}

func (fp FeePolicy) CalculateOriginalAmount(amount *uint256.Int, reverseFee bool) *uint256.Int {
	r := fp.Rate()
	if reverseFee {
		r = -r
	}

	var divider uint256.Int
	if r <= 0 {
		divider.Sub(ratePrecision, uint256.NewInt(uint64(r)))
	} else {
		divider.Add(ratePrecision, uint256.NewInt(uint64(r)))
	}

	originalAmount := new(uint256.Int)
	return u256.MulDivRounding(originalAmount, amount, ratePrecision, &divider, reverseFee)
}
