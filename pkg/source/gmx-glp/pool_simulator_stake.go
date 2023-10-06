package gmxglp

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"math/big"
)

func (p *PoolSimulator) MintAndStakeGlp(tokenIn string, amount *big.Int) (*big.Int, error) {
	if amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrRewardRouterInvalidAmount
	}
	glpAmount, err := p.addliquidityForAccount(tokenIn, amount)
	if err != nil {
		return nil, err
	}

	return glpAmount, nil
}

func (p *PoolSimulator) addliquidityForAccount(tokenIn string, amount *big.Int) (*big.Int, error) {
	// _addLiquidity
	if amount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrGlpManagerInvalidAmount
	}

	// calculate aum before buyUSDG
	aumInUsdg := new(big.Int).Set(p.glpManager.MaximiseAumInUsdg)
	glpSupply := new(big.Int).Set(p.glpManager.GlpTotalSupply)

	usdgAmount, err := p.BuyUSDG(tokenIn, amount)
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

func (p *PoolSimulator) BuyUSDG(tokenIn string, tokenAmount *big.Int) (*big.Int, error) {
	//_validate(whitelistedTokens[_token], 16);  // canSwapTo vaildated it
	p.vault.UseSwapPricing = true

	// uint256 tokenAmount = _transferIn(_token);

	// _validate(tokenAmount > 0, 17);
	if tokenAmount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrVaultNegativeTokenAmount
	}

	// updateCumulativeFundingRate(_token, _token);

	price, err := p.vault.GetMinPrice(tokenIn)
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
	usdgAmount = p.vault.AdjustForDecimals(usdgAmount, tokenIn, p.vault.USDG.Address)
	if usdgAmount.Cmp(bignumber.ZeroBI) <= 0 {
		return nil, ErrVaultNegativeUsdgAmount
	}

	// getBuyUsdgFeeBasisPoints
	feeBasicPoints := p.vaultUtils.GetBuyUsdgFeeBasisPoints(tokenIn, usdgAmount)
	amountAfterFees, err := p.vault.CollectSwapFees(tokenIn, tokenAmount, feeBasicPoints)
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
	mintAmount = p.vault.AdjustForDecimals(mintAmount, tokenIn, p.vault.USDG.Address)

	p.vault.IncreaseUSDGAmount(tokenIn, mintAmount)
	p.vault.IncreasePoolAmount(tokenIn, amountAfterFees)

	//IUSDG(usdg).mint(_receiver, mintAmount);

	p.vault.UseSwapPricing = false

	return mintAmount, nil
}
