package dpp

import (
	"fmt"

	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/dodo/libv2"
)

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/DODOPrivatePool/impl/DPPTrader.sol#L201
func (p *PoolSimulator) querySellBase(payBaseAmount *uint256.Int) (
	receiveQuoteAmount *uint256.Int,
	lpFee *uint256.Int,
	mtFee *uint256.Int,
	err error,
) {
	defer func() {
		if r := recover(); r != nil {
			if recoveredError, ok := r.(error); ok {
				err = recoveredError
			} else {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
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

// https://github.com/DODOEX/contractV2/blob/c58c067c4038437610a9cc8aef8f8025e2af4f63/contracts/DODOPrivatePool/impl/DPPTrader.sol#L223
func (p *PoolSimulator) querySellQuote(payQuoteAmount *uint256.Int) (
	receiveBaseAmount *uint256.Int,
	lpFee *uint256.Int,
	mtFee *uint256.Int,
	err error,
) {
	defer func() {
		if r := recover(); r != nil {
			if recoveredError, ok := r.(error); ok {
				err = recoveredError
			} else {
				err = fmt.Errorf("unexpected panic: %v", r)
			}
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
