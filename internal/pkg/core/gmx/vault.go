package gmx

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"

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
	USDGAmounts       map[string]*big.Int `json:"usdgAmounts"`
	MaxUSDGAmounts    map[string]*big.Int `json:"maxUsdgAmounts"`
	TokenWeights      map[string]*big.Int `json:"tokenWeights"`

	PriceFeed *VaultPriceFeed `json:"priceFeed"`
	USDG      *USDG           `json:"usdg"`
}

func (v *Vault) GetMinPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, false, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetMaxPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, true, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetTargetUSDGAmount(token string) *big.Int {
	supply := v.USDG.TotalSupply

	if supply.Cmp(constant.Zero) == 0 {
		return constant.Zero
	}

	weight := v.TokenWeights[token]

	return new(big.Int).Div(new(big.Int).Mul(weight, supply), v.TotalTokenWeights)
}

func (v *Vault) AdjustForDecimals(amount *big.Int, tokenDiv string, tokenMul string) *big.Int {
	var decimalsDiv *big.Int
	if tokenDiv == v.USDG.Address {
		decimalsDiv = USDGDecimals
	} else {
		decimalsDiv = v.TokenDecimals[tokenDiv]
	}

	var decimalsMul *big.Int
	if tokenMul == v.USDG.Address {
		decimalsMul = USDGDecimals
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

func (v *Vault) IncreaseUSDGAmount(token string, amount *big.Int) {
	v.USDGAmounts[token] = new(big.Int).Add(v.USDGAmounts[token], amount)
}

func (v *Vault) DecreaseUSDGAmount(token string, amount *big.Int) {
	currentUSDGAmount := v.USDGAmounts[token]

	if currentUSDGAmount.Cmp(amount) < 0 {
		v.USDGAmounts[token] = constant.Zero
		return
	}

	v.USDGAmounts[token] = new(big.Int).Sub(v.USDGAmounts[token], amount)
}

func (v *Vault) IncreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Add(v.PoolAmounts[token], amount)
}

func (v *Vault) DecreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Sub(v.PoolAmounts[token], amount)
}
