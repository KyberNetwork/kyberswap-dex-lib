package metavault

import (
	"github.com/KyberNetwork/router-service/internal/pkg/constant"

	"math/big"
)

// Vault
// https://github.com/gmx-io/gmx-contracts/blob/master/contracts/core/Vault.sol
type Vault struct {
	HasDynamicFees           bool     `json:"hasDynamicFees"`
	IncludeAmmPrice          bool     `json:"includeAmmPrice"`
	IsSwapEnabled            bool     `json:"isSwapEnabled"`
	StableSwapFeeBasisPoints *big.Int `json:"stableSwapFeeBasisPoints"`
	StableTaxBasisPoints     *big.Int `json:"stableTaxBasisPoints"`
	SwapFeeBasisPoints       *big.Int `json:"swapFeeBasisPoints"`
	TaxBasisPoints           *big.Int `json:"taxBasisPoints"`
	TotalTokenWeights        *big.Int `json:"totalTokenWeights"`
	UseSwapPricing           bool     `json:"useSwapPricing"`

	WhitelistedTokens []string            `json:"whitelistedTokens"`
	PoolAmounts       map[string]*big.Int `json:"poolAmounts"`
	ReservedAmounts   map[string]*big.Int `json:"reservedAmounts"`
	BufferAmounts     map[string]*big.Int `json:"bufferAmounts"`
	TokenDecimals     map[string]*big.Int `json:"tokenDecimals"`
	StableTokens      map[string]bool     `json:"stableTokens"`
	USDMAmounts       map[string]*big.Int `json:"usdmAmounts"`
	MaxUSDMAmounts    map[string]*big.Int `json:"maxUsdmAmounts"`
	TokenWeights      map[string]*big.Int `json:"tokenWeights"`

	PriceFeed *VaultPriceFeed `json:"priceFeed"`
	USDM      *USDM           `json:"usdm"`
}

func (v *Vault) GetMinPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, false, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetMaxPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, true, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetTargetUSDMAmount(token string) *big.Int {
	supply := v.USDM.TotalSupply

	if supply.Cmp(constant.Zero) == 0 {
		return constant.Zero
	}

	weight := v.TokenWeights[token]

	return new(big.Int).Div(new(big.Int).Mul(weight, supply), v.TotalTokenWeights)
}

func (v *Vault) AdjustForDecimals(amount *big.Int, tokenDiv string, tokenMul string) *big.Int {
	var decimalsDiv *big.Int
	if tokenDiv == v.USDM.Address {
		decimalsDiv = USDMDecimals
	} else {
		decimalsDiv = v.TokenDecimals[tokenDiv]
	}

	var decimalsMul *big.Int
	if tokenMul == v.USDM.Address {
		decimalsMul = USDMDecimals
	} else {
		decimalsMul = v.TokenDecimals[tokenMul]
	}

	return new(big.Int).Div(
		new(big.Int).Mul(
			amount,
			new(big.Int).Exp(big.NewInt(10), decimalsMul, nil),
		),
		new(big.Int).Exp(big.NewInt(10), decimalsDiv, nil),
	)
}

func (v *Vault) IncreaseUSDMAmount(token string, amount *big.Int) {
	v.USDMAmounts[token] = new(big.Int).Add(v.USDMAmounts[token], amount)
}

func (v *Vault) DecreaseUSDMAmount(token string, amount *big.Int) {
	currentUSDMAmount := v.USDMAmounts[token]

	if currentUSDMAmount.Cmp(amount) < 0 {
		v.USDMAmounts[token] = constant.Zero
		return
	}

	v.USDMAmounts[token] = new(big.Int).Sub(v.USDMAmounts[token], amount)
}

func (v *Vault) IncreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Add(v.PoolAmounts[token], amount)
}

func (v *Vault) DecreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Sub(v.PoolAmounts[token], amount)
}
