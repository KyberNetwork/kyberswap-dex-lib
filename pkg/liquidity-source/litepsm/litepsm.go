package litepsm

import (
	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/holiman/uint256"
)

// PSM implements DssPsm contract
// https://github.com/makerdao/dss-psm/blob/master/src/psm.sol

func (lpsm *LitePSM) sellGem(
	gemAmt *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	if lpsm.TIn.Cmp(HALTED) == 0 {
		return number.Zero, number.Zero, ErrSellGemHalted
	}

	daiOutWad := new(uint256.Int).Mul(gemAmt, lpsm.To18ConversionFactor)
	fee := new(uint256.Int)

	if lpsm.TIn.Cmp(number.Zero) > 0 {
		fee.Mul(daiOutWad, lpsm.TIn)
		fee.Div(fee, WAD)
		daiOutWad.Sub(daiOutWad, fee)
	}

	// On the contract, we have this transfer: `dai.transfer(usr, daiOutWad)`;
	// We need to make sure that this transfer will not fail -> the daiOutWad must be less than or equal to the balance of the contract
	if daiOutWad.Cmp(lpsm.DaiBalance) > 0 {
		return number.Zero, number.Zero, ErrInsufficientDAIBalance
	}

	return daiOutWad, fee, nil
}

func (lpsm *LitePSM) buyGem(
	daiAmt *uint256.Int,
) (*uint256.Int, *uint256.Int, error) {
	if lpsm.TOut.Cmp(HALTED) == 0 {
		return number.Zero, number.Zero, ErrBuyGemHalted
	}

	daiInWad := new(uint256.Int).Set(daiAmt)
	fee := new(uint256.Int)

	if lpsm.TOut.Cmp(number.Zero) > 0 {
		// Calculate fee
		fee.Mul(daiInWad, lpsm.TOut)
		fee.Div(fee, WAD)

		// Subtract fee from daiInWad
		daiInWad.Sub(daiInWad, fee)
	}

	// Convert daiInWad to gemAmt
	gemAmt := new(uint256.Int).Div(daiInWad, lpsm.To18ConversionFactor)

	// On the contract, we have this transfer: `gem.transferFrom(pocket, usr, gemAmt)`;
	// We need to make sure that this transfer will not fail -> the gemAmt must be less than or equal to the balance of the pocket
	if gemAmt.Cmp(lpsm.GemBalance) > 0 {
		return number.Zero, number.Zero, ErrInsufficientGemBalance
	}

	return gemAmt, fee, nil
}

func (lpsm *LitePSM) updateBalanceSellingGem(gemAmt, daiAmt *uint256.Int) {
	lpsm.DaiBalance.Sub(lpsm.DaiBalance, daiAmt)
	lpsm.GemBalance.Add(lpsm.GemBalance, gemAmt)
}

func (lpsm *LitePSM) updateBalanceBuyingGem(daiAmt, gemAmt *uint256.Int) {
	lpsm.DaiBalance.Add(lpsm.DaiBalance, daiAmt)
	lpsm.GemBalance.Sub(lpsm.GemBalance, gemAmt)
}
