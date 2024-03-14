package stablemetang

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/holiman/uint256"
)

// old pools still follow this interface, we'll convert them to new one
type ICurveBasePoolLegacy interface {
	GetInfo() pool.PoolInfo
	GetTokenIndex(address string) int
	// return both vPrice and D
	GetVirtualPrice() (vPrice *big.Int, D *big.Int, err error)
	CalculateTokenAmount(amounts []*big.Int, deposit bool) (*big.Int, error)
	CalculateWithdrawOneCoin(tokenAmount *big.Int, i int) (*big.Int, *big.Int, error)
	AddLiquidity(amounts []*big.Int) (*big.Int, error)
	RemoveLiquidityOneCoin(tokenAmount *big.Int, i int) (*big.Int, error)
}

type legacyWrapper struct{ ICurveBasePoolLegacy }

func (w legacyWrapper) GetVirtualPriceU256(vPrice *uint256.Int, D *uint256.Int) error {
	vPriceBI, DBI, err := w.GetVirtualPrice()
	vPrice.SetFromBig(vPriceBI)
	D.SetFromBig(DBI)
	return err
}

func (w legacyWrapper) CalculateTokenAmountU256(amounts []uint256.Int, deposit bool, mintAmount *uint256.Int, feeAmounts []uint256.Int) error {
	amountsBI := make([]*big.Int, len(amounts))
	for i := range amounts {
		amountsBI[i] = amounts[i].ToBig()
	}

	mintAmountBI, err := w.CalculateTokenAmount(amountsBI, deposit)
	if err != nil {
		return err
	}

	mintAmount.SetFromBig(mintAmountBI)

	return nil
}

func (w legacyWrapper) CalculateWithdrawOneCoinU256(tokenAmount *uint256.Int, i int, dy *uint256.Int, dyFee *uint256.Int) error {
	dyBI, dyFeeBI, err := w.CalculateWithdrawOneCoin(tokenAmount.ToBig(), i)
	if err != nil {
		return err
	}

	dy.SetFromBig(dyBI)
	dyFee.SetFromBig(dyFeeBI)
	return nil
}

func (w legacyWrapper) ApplyRemoveLiquidityOneCoinU256(i int, tokenAmount, dy, dyFee *uint256.Int) error {
	_, err := w.RemoveLiquidityOneCoin(tokenAmount.ToBig(), i)
	return err
}

func (w legacyWrapper) ApplyAddLiquidity(amounts, feeAmounts []uint256.Int, mintAmount *uint256.Int) error {
	amountsBI := make([]*big.Int, len(amounts))
	for i := range amounts {
		amountsBI[i] = amounts[i].ToBig()
	}

	_, err := w.AddLiquidity(amountsBI)
	return err
}

type ICurveBasePoolLegacy2 interface {
	GetInfo() pool.PoolInfo
	GetTokenIndex(address string) int

	GetVirtualPriceU256(vPrice, D *uint256.Int) error

	CalculateTokenAmountU256(
		amounts []uint256.Int,
		deposit bool,
	) (*uint256.Int, error)
	// CalculateWithdrawOneCoinU256(
	// 	tokenAmount *uint256.Int,
	// 	i int,
	// ) (*uint256.Int, *uint256.Int, error)
	CalculateWithdrawOneCoinU256(
		tokenAmount *uint256.Int,
		i int,

		// output
		dy *uint256.Int, dyFee *uint256.Int,
	) error
	AddLiquidityU256(amounts []uint256.Int) (*uint256.Int, error)
	RemoveLiquidityOneCoinU256(tokenAmount *uint256.Int, i int) (*uint256.Int, error)

	ApplyRemoveLiquidityOneCoinU256(i int, tokenAmount, dy, dyFee *uint256.Int) error
}

type legacyWrapper2 struct{ ICurveBasePoolLegacy2 }

// func (w legacyWrapper2) GetVirtualPriceU256(vPrice *uint256.Int, D *uint256.Int) (err error) {
// 	_vPrice, _D, err := w.ICurveBasePoolLegacy2.GetVirtualPriceU256()
// 	if err != nil {
// 		return
// 	}
// 	vPrice.Set(_vPrice)
// 	D.Set(_D)
// 	return
// }

func (w legacyWrapper2) CalculateTokenAmountU256(amounts []uint256.Int, deposit bool, mintAmount *uint256.Int, feeAmounts []uint256.Int) (err error) {
	_mintAmount, err := w.ICurveBasePoolLegacy2.CalculateTokenAmountU256(amounts, deposit)
	if err != nil {
		return
	}
	mintAmount.Set(_mintAmount)
	return
}

// func (w legacyWrapper2) CalculateWithdrawOneCoinU256(tokenAmount *uint256.Int, i int, dy *uint256.Int, dyFee *uint256.Int) (err error) {
// 	_dy, _dyFee, err := w.ICurveBasePoolLegacy2.CalculateWithdrawOneCoinU256(tokenAmount, i)
// 	if err != nil {
// 		return
// 	}
// 	dy.Set(_dy)
// 	dyFee.Set(_dyFee)
// 	return
// }

// func (w legacyWrapper2) ApplyRemoveLiquidityOneCoinU256(i int, tokenAmount, dy, dyFee *uint256.Int) error {
// 	_, err := w.RemoveLiquidityOneCoinU256(tokenAmount, i)
// 	return err
// }

func (w legacyWrapper2) ApplyAddLiquidity(amounts, feeAmounts []uint256.Int, mintAmount *uint256.Int) error {
	_, err := w.AddLiquidityU256(amounts)
	return err
}
