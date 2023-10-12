package gmxglp

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
)

func (p *PoolSimulator) UnstakeAndRedeemGlp(tokenOut string, glpAmount *big.Int) (*big.Int, error) {
	if glpAmount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrRewardRouterInvalidGlpAmount
	}

	amountOut, err := p.removeLiquidityForAccount(tokenOut, glpAmount)
	if err != nil {
		return nil, err
	}

	return amountOut, nil
}

func (p *PoolSimulator) removeLiquidityForAccount(tokenOut string, glpAmount *big.Int) (*big.Int, error) {
	if glpAmount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrGlpManagerInvalidAmount
	}

	aumInUsdg := new(big.Int).Set(p.glpManager.NotMaximiseAumInUsdg)
	glpSupply := new(big.Int).Set(p.glpManager.GlpTotalSupply)

	usdgAmount, err := mul(glpAmount, aumInUsdg)
	if err != nil {
		return nil, err
	}
	usdgAmount, err = div(usdgAmount, glpSupply)
	if err != nil {
		return nil, err
	}

	//uint256 usdgBalance = IERC20(usdg).balanceOf(address(this));
	//if (usdgAmount > usdgBalance) {
	//	IUSDG(usdg).mint(address(this), usdgAmount.sub(usdgBalance));
	//}
	//IMintable(glp).burn(_account, _glpAmount);
	//IERC20(usdg).transfer(address(vault), usdgAmount);

	amountOut, err := p.SellUSDG(tokenOut, usdgAmount)
	if err != nil {
		return nil, err
	}

	return amountOut, nil
}

func (p *PoolSimulator) SellUSDG(token string, usdgAmount *big.Int) (*big.Int, error) {
	//_validate(whitelistedTokens[_token], 19);  // handled at canSwapTo
	p.vault.UseSwapPricing = true

	if usdgAmount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrVaultNegativeUsdgAmount
	}

	redemptionAmount, err := p.getRedemptionAmount(token, usdgAmount)
	if err != nil {
		return nil, err
	}
	if redemptionAmount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrVaultNegativeRedemptionAmount
	}

	p.swapInfo.usdgAmount = new(big.Int).Set(usdgAmount)
	p.swapInfo.redemptionAmount = new(big.Int).Set(redemptionAmount)
	//p.vault.DecreaseUSDGAmount(token, usdgAmount)
	//p.vault.DecreasePoolAmount(token, redemptionAmount)
	// updateTokenBalance(usdg)

	feeBasisPoints := p.vaultUtils.GetSellUsdgFeeBasisPoints(token, usdgAmount)
	amountOut, err := p.vault.CollectSwapFees(token, redemptionAmount, feeBasisPoints)
	if err != nil {
		return nil, err
	}
	if amountOut.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrVaultNegativeAmountOut
	}

	p.vault.UseSwapPricing = false

	return amountOut, nil
}

func (p *PoolSimulator) getRedemptionAmount(token string, usdgAmount *big.Int) (*big.Int, error) {
	price, err := p.vault.GetMaxPrice(token)
	if err != nil {
		return nil, err
	}
	redemptionAmount, err := mul(usdgAmount, PricePrecision)
	if err != nil {
		return nil, err
	}
	redemptionAmount, err = div(redemptionAmount, price)
	if err != nil {
		return nil, err
	}

	return p.vault.AdjustForDecimals(redemptionAmount, p.vault.USDG.Address, token), nil
}
