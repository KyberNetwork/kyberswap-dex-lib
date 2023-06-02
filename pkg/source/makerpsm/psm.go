package makerpsm

import (
	"math/big"
)

// PSM implements DssPsm contract
// https://github.com/makerdao/dss-psm/blob/master/src/psm.sol

func (psm *PSM) sellGem(
	gemAmt *big.Int,
) (*big.Int, *big.Int, error) {
	// gemAmt18 = amount * to18ConversionFactor
	gemAmt18 := new(big.Int).Mul(gemAmt, psm.To18ConversionFactor)
	// fee = gemAmt18 * tin / WAD
	fee := new(big.Int).Div(new(big.Int).Mul(gemAmt18, psm.TIn), WAD)
	// daiAmt = gemAmt18 - fee
	daiAmt := new(big.Int).Sub(gemAmt18, fee)

	if err := psm.Vat.validateSellingGem(gemAmt18); err != nil {
		return nil, nil, err
	}

	return daiAmt, fee, nil
}

func (psm *PSM) buyGem(
	daiAmt *big.Int,
) (*big.Int, *big.Int, error) {
	gemAmt18 := new(big.Int).Div(
		new(big.Int).Mul(daiAmt, WAD),
		new(big.Int).Add(psm.TOut, WAD),
	)

	if err := psm.Vat.validateBuyingGem(gemAmt18); err != nil {
		return nil, nil, err
	}

	fee := new(big.Int).Sub(
		daiAmt,
		gemAmt18,
	)

	gemAmt := new(big.Int).Div(gemAmt18, psm.To18ConversionFactor)

	return gemAmt, fee, nil
}

func (psm *PSM) updateBalanceSellingGem(gemAmt *big.Int) {
	gemAmt18 := new(big.Int).Mul(gemAmt, psm.To18ConversionFactor)

	psm.Vat.updateBalanceSellingGem(gemAmt18)
}

func (psm *PSM) updateBalanceBuyingGem(daiAmt *big.Int) {
	gemAmt18 := new(big.Int).Div(
		new(big.Int).Mul(daiAmt, WAD),
		new(big.Int).Add(psm.TOut, WAD),
	)

	psm.Vat.updateBalanceBuyingGem(gemAmt18)
}
