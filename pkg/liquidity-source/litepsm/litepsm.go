package litepsm

import (
	"github.com/holiman/uint256"

	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

// PSM implements DssPsm contract
// https://github.com/makerdao/dss-psm/blob/master/src/psm.sol

func (p *PoolSimulator) sellGem(gemAmt *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	if p.TIn != nil && p.TIn.Cmp(HALTED) == 0 {
		return nil, nil, ErrSellGemHalted
	}

	daiOutWad := new(uint256.Int).Mul(gemAmt, p.To18ConversionFactor)
	fee := new(uint256.Int)

	if p.TIn != nil && p.TIn.Sign() > 0 {
		daiOutWad.Sub(daiOutWad, big256.MulWadDown(fee, daiOutWad, p.TIn))
	}

	// On the contract, we have this transfer: `dai.transfer(usr, daiOutWad)`;
	// We need to make sure that this transfer will not fail -> the daiOutWad must be less than or equal to the balance of the contract
	if daiOutWad.Cmp(p.DaiBal) > 0 {
		return nil, nil, ErrInsufficientDAIBalance
	}

	return daiOutWad, fee, nil
}

func (p *PoolSimulator) buyGem(daiAmt *uint256.Int) (*uint256.Int, *uint256.Int, error) {
	if p.TOut != nil && p.TOut.Cmp(HALTED) == 0 {
		return nil, nil, ErrBuyGemHalted
	}

	daiInWad := new(uint256.Int).Set(daiAmt)
	fee := new(uint256.Int)

	if p.TOut != nil && p.TOut.Sign() > 0 {
		// Calculate and subtract fee from daiInWad
		daiInWad.Sub(daiInWad, big256.MulWadDown(fee, daiInWad, p.TOut))
	}

	// Convert daiInWad to gemAmt
	gemAmt := daiInWad.Div(daiInWad, p.To18ConversionFactor)

	// On the contract, we have this transfer: `gem.transferFrom(pocket, usr, gemAmt)`;
	// We need to make sure that this transfer will not fail -> the gemAmt must be less than or equal to the balance of the pocket
	if gemAmt.Cmp(p.GemBal) > 0 {
		return nil, nil, ErrInsufficientGemBalance
	}

	return gemAmt, fee, nil
}

func (p *PoolSimulator) updateBalanceSellingGem(gemAmt, daiAmt *uint256.Int) {
	p.DaiBal = daiAmt.Sub(p.DaiBal, daiAmt)
	p.GemBal = gemAmt.Add(p.GemBal, gemAmt)
}

func (p *PoolSimulator) updateBalanceBuyingGem(daiAmt, gemAmt *uint256.Int) {
	p.DaiBal = daiAmt.Add(p.DaiBal, daiAmt)
	p.GemBal = gemAmt.Sub(p.GemBal, gemAmt)
}
