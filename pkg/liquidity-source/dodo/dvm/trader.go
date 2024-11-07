package dvm

import (
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/libv2"
)

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/DODOVendingMachine/impl/DVMTrader.sol#L151
func (p *PoolSimulator) querySellBase(payBaseAmount *uint256.Int) (
	receiveQuoteAmount *uint256.Int,
	lpFee *uint256.Int,
	mtFee *uint256.Int,
	err error,
) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	receiveQuoteAmount, _ = libv2.SellBaseToken(p.getPMMState(), payBaseAmount)
	lpFee = libv2.DecimalMathMulFloor(receiveQuoteAmount, p.LpFeeRate)
	mtFee = libv2.DecimalMathMulFloor(receiveQuoteAmount, p.MtFeeRate)
	receiveQuoteAmount = libv2.SafeSub(
		libv2.SafeSub(
			receiveQuoteAmount,
			lpFee,
		),
		mtFee,
	)

	return
}

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/DODOVendingMachine/impl/DVMTrader.sol#L166
func (p *PoolSimulator) querySellQuote(payQuoteAmount *uint256.Int) (
	receiveBaseAmount *uint256.Int,
	lpFee *uint256.Int,
	mtFee *uint256.Int,
	err error,
) {
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)
		}
	}()

	receiveBaseAmount, _ = libv2.SellQuoteToken(p.getPMMState(), payQuoteAmount)
	lpFee = libv2.DecimalMathMulFloor(receiveBaseAmount, p.LpFeeRate)
	mtFee = libv2.DecimalMathMulFloor(receiveBaseAmount, p.MtFeeRate)
	receiveBaseAmount = libv2.SafeSub(
		libv2.SafeSub(
			receiveBaseAmount,
			lpFee,
		),
		mtFee,
	)

	return
}
