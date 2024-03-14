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
	// if `dCached` is nil then will be recalculated
	GetDy(i int, j int, dx *big.Int, dCached *big.Int) (*big.Int, *big.Int, error)
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
