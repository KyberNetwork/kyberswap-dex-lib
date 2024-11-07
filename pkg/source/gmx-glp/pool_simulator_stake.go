package gmxglp

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func (p *PoolSimulator) MintAndStakeGlp(swapInfo *gmxGlpSwapInfo, tokenIn string, amount *big.Int) (*big.Int, error) {
	if amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrRewardRouterInvalidAmount
	}
	glpAmount, err := p.addLiquidityForAccount(swapInfo, tokenIn, amount)
	if err != nil {
		return nil, err
	}

	return glpAmount, nil
}

func (p *PoolSimulator) addLiquidityForAccount(swapInfo *gmxGlpSwapInfo, tokenIn string, amount *big.Int) (*big.Int, error) {
	// _addLiquidity
	if amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrGlpManagerInvalidAmount
	}

	// calculate aum before buyUSDG
	aumInUsdg := new(big.Int).Set(p.glpManager.MaximiseAumInUsdg)
	glpSupply := new(big.Int).Set(p.glpManager.GlpTotalSupply)

	usdgAmount, err := p.BuyUSDG(swapInfo, tokenIn, amount)
	if err != nil {
		return nil, err
	}

	var mintAmount *big.Int
	if aumInUsdg.Cmp(bignumber.ZeroBI) == 0 {
		mintAmount = new(big.Int).Set(usdgAmount)
	} else {
		tmp, err := mul(usdgAmount, glpSupply)
		if err != nil {
			return nil, err
		}
		mintAmount, err = div(tmp, aumInUsdg)
		if err != nil {
			return nil, err
		}
	}

	// IMintable(glp).mint(_account, mintAmount);
	// lastAddedAt[_account] = block.timestamp;

	return mintAmount, nil
}

func (p *PoolSimulator) BuyUSDG(swapInfo *gmxGlpSwapInfo, token string, tokenAmount *big.Int) (*big.Int, error) {
	//_validate(whitelistedTokens[_token], 16);  // canSwapTo vaildated it
	useSwapPricing := true

	// uint256 tokenAmount = _transferIn(_token);

	// _validate(tokenAmount > 0, 17);
	if tokenAmount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrVaultNegativeTokenAmount
	}

	// updateCumulativeFundingRate(_token, _token);

	price, err := p.vault.GetMinPrice(token, useSwapPricing)
	if err != nil {
		return nil, err
	}

	usdgAmount, err := mul(tokenAmount, price)
	if err != nil {
		return nil, err
	}
	usdgAmount, err = div(usdgAmount, PricePrecision)
	if err != nil {
		return nil, err
	}
	usdgAmount = p.vault.AdjustForDecimals(usdgAmount, token, p.vault.USDG.Address)
	if usdgAmount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrVaultNegativeUsdgAmount
	}

	// getBuyUsdgFeeBasisPoints
	feeBasicPoints := p.vaultUtils.GetBuyUsdgFeeBasisPoints(token, usdgAmount)
	amountAfterFees, err := p.vault.CollectSwapFees(token, tokenAmount, feeBasicPoints)
	if err != nil {
		return nil, err
	}
	mintAmount, err := mul(amountAfterFees, price)
	if err != nil {
		return nil, err
	}
	mintAmount, err = div(mintAmount, PricePrecision)
	if err != nil {
		return nil, err
	}
	mintAmount = p.vault.AdjustForDecimals(mintAmount, token, p.vault.USDG.Address)

	// swapInfo for caching result to updateBalance
	swapInfo.mintAmount = new(big.Int).Set(mintAmount)
	swapInfo.amountAfterFees = new(big.Int).Set(amountAfterFees)
	//p.vault.IncreaseUSDGAmount(tokenIn, mintAmount)
	if err = p.validateMaxUsdgExceed(token, mintAmount); err != nil {
		return nil, err
	}
	//p.vault.IncreasePoolAmount(tokenIn, amountAfterFees)

	//IUSDG(usdg).mint(_receiver, mintAmount);

	return mintAmount, nil
}
