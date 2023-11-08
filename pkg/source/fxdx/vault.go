package fxdx

import (
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/ethereum/go-ethereum/common"
)

type Vault struct {
	IncludeAmmPrice   bool     `json:"includeAmmPrice"`
	IsSwapEnabled     bool     `json:"isSwapEnabled"`
	TotalTokenWeights *big.Int `json:"totalTokenWeights"`

	WhitelistedTokens      []string `json:"whitelistedTokens"`
	WhitelistedTokensCount *big.Int `json:"-"`

	PoolAmounts     map[string]*big.Int `json:"poolAmounts"`
	BufferAmounts   map[string]*big.Int `json:"bufferAmounts"`
	ReservedAmounts map[string]*big.Int `json:"reservedAmounts"`
	TokenDecimals   map[string]*big.Int `json:"tokenDecimals"`
	StableTokens    map[string]bool     `json:"stableTokens"`
	USDFAmounts     map[string]*big.Int `json:"usdfAmounts"`
	MaxUSDFAmounts  map[string]*big.Int `json:"maxUsdfAmounts"`
	TokenWeights    map[string]*big.Int `json:"tokenWeights"`

	PriceFeedAddress common.Address  `json:"-"`
	PriceFeed        *VaultPriceFeed `json:"priceFeed"`

	USDFAddress common.Address `json:"-"`
	USDF        *USDF          `json:"usdf"`

	UseSwapPricing bool `json:"useSwapPricing"` // not used, always false for now

	FeeUtils common.Address `json:"-"`
}

func NewVault() *Vault {
	return &Vault{
		PoolAmounts:     make(map[string]*big.Int),
		BufferAmounts:   make(map[string]*big.Int),
		ReservedAmounts: make(map[string]*big.Int),
		TokenDecimals:   make(map[string]*big.Int),
		StableTokens:    make(map[string]bool),
		USDFAmounts:     make(map[string]*big.Int),
		MaxUSDFAmounts:  make(map[string]*big.Int),
		TokenWeights:    make(map[string]*big.Int),
	}
}

const (
	vaultMethodHasDynamicFees           = "hasDynamicFees"
	vaultMethodIncludeAmmPrice          = "includeAmmPrice"
	vaultMethodIsSwapEnabled            = "isSwapEnabled"
	vaultMethodPriceFeed                = "priceFeed"
	vaultMethodStableSwapFeeBasisPoints = "stableSwapFeeBasisPoints"
	vaultMethodStableTaxBasisPoints     = "stableTaxBasisPoints"
	vaultMethodSwapFeeBasisPoints       = "swapFeeBasisPoints"
	vaultMethodTaxBasisPoints           = "taxBasisPoints"
	vaultMethodTotalTokenWeights        = "totalTokenWeights"
	vaultMethodUSDF                     = "usdf"
	vaultMethodWhitelistedTokenCount    = "whitelistedTokenCount"

	vaultMethodAllWhitelistedTokens = "allWhitelistedTokens"

	vaultMethodPoolAmounts     = "poolAmounts"
	vaultMethodBufferAmounts   = "bufferAmounts"
	vaultMethodReservedAmounts = "reservedAmounts"
	vaultMethodTokenDecimals   = "tokenDecimals"
	vaultMethodStableTokens    = "stableTokens"
	vaultMethodUSDFAmounts     = "usdfAmounts"
	vaultMethodMaxUSDFAmounts  = "maxUsdfAmounts"
	vaultMethodTokenWeights    = "tokenWeights"

	vaultMethodFeeUtils = "feeUtils"
)

func (v *Vault) GetMinPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, false, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetMaxPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, true, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) AdjustForDecimals(amount *big.Int, tokenDiv string, tokenMul string) *big.Int {
	var decimalsDiv *big.Int
	if tokenDiv == v.USDF.Address {
		decimalsDiv = USDFDecimals
	} else {
		decimalsDiv = v.TokenDecimals[tokenDiv]
	}

	var decimalsMul *big.Int
	if tokenMul == v.USDF.Address {
		decimalsMul = USDFDecimals
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

func (v *Vault) GetTargetUSDFAmount(token string) *big.Int {
	supply := v.USDF.TotalSupply

	if supply.Cmp(integer.Zero()) == 0 {
		return integer.Zero()
	}

	weight := v.TokenWeights[token]

	return new(big.Int).Div(new(big.Int).Mul(weight, supply), v.TotalTokenWeights)
}

func (v *Vault) IncreaseUSDFAmount(token string, amount *big.Int) {
	v.USDFAmounts[token] = new(big.Int).Add(v.USDFAmounts[token], amount)
}

func (v *Vault) DecreaseUSDFAmount(token string, amount *big.Int) {
	currentUSDFAmount := v.USDFAmounts[token]

	if currentUSDFAmount.Cmp(amount) < 0 {
		v.USDFAmounts[token] = integer.Zero()
		return
	}

	v.USDFAmounts[token] = new(big.Int).Sub(v.USDFAmounts[token], amount)
}

func (v *Vault) IncreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Add(v.PoolAmounts[token], amount)
}

func (v *Vault) DecreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Sub(v.PoolAmounts[token], amount)
}
