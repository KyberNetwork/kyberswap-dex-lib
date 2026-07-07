package gmx

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type Vault struct {
	HasDynamicFees           bool     `json:"hasDynamicFees,omitempty"`
	IncludeAmmPrice          bool     `json:"includeAmmPrice,omitempty"`
	IsSwapEnabled            bool     `json:"isSwapEnabled,omitempty"`
	StableSwapFeeBasisPoints *big.Int `json:"stableSwapFeeBasisPoints,omitempty"`
	StableTaxBasisPoints     *big.Int `json:"stableTaxBasisPoints,omitempty"`
	SwapFeeBasisPoints       *big.Int `json:"swapFeeBasisPoints,omitempty"`
	TaxBasisPoints           *big.Int `json:"taxBasisPoints,omitempty"`
	TotalTokenWeights        *big.Int `json:"totalTokenWeights,omitempty"`

	WhitelistedTokens []string            `json:"whitelistedTokens,omitempty"`
	PoolAmounts       map[string]*big.Int `json:"poolAmounts,omitempty"`
	BufferAmounts     map[string]*big.Int `json:"bufferAmounts,omitempty"`
	ReservedAmounts   map[string]*big.Int `json:"reservedAmounts,omitempty"`
	TokenDecimals     map[string]*big.Int `json:"tokenDecimals,omitempty"`
	StableTokens      map[string]bool     `json:"stableTokens,omitempty"`
	USDGAmounts       map[string]*big.Int `json:"usdgAmounts,omitempty"`
	MaxUSDGAmounts    map[string]*big.Int `json:"maxUsdgAmounts,omitempty"`
	TokenWeights      map[string]*big.Int `json:"tokenWeights,omitempty"`

	PriceFeedAddress common.Address  `json:"-"`
	PriceFeed        *VaultPriceFeed `json:"priceFeed,omitempty"`

	USDGAddress common.Address `json:"-"`
	USDG        *USDG          `json:"usdg,omitempty"`

	WhitelistedTokensCount *big.Int `json:"-"`

	UseSwapPricing bool `json:"useSwapPricing,omitempty"` // not used, always false for now
}

func NewVault() *Vault {
	return &Vault{
		PoolAmounts:     make(map[string]*big.Int),
		BufferAmounts:   make(map[string]*big.Int),
		ReservedAmounts: make(map[string]*big.Int),
		TokenDecimals:   make(map[string]*big.Int),
		StableTokens:    make(map[string]bool),
		USDGAmounts:     make(map[string]*big.Int),
		MaxUSDGAmounts:  make(map[string]*big.Int),
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

	vaultMethodAllWhitelistedTokensLength = "allWhitelistedTokensLength"
	vaultMethodAllWhitelistedTokens       = "allWhitelistedTokens"
	vaultMethodWhitelistedTokens          = "whitelistedTokens"

	vaultMethodPoolAmounts     = "poolAmounts"
	vaultMethodBufferAmounts   = "bufferAmounts"
	vaultMethodReservedAmounts = "reservedAmounts"
	vaultMethodTokenDecimals   = "tokenDecimals"
	vaultMethodStableTokens    = "stableTokens"
	vaultMethodTokenWeights    = "tokenWeights"
)

func (v *Vault) GetMinPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, false, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetMaxPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, true, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetTargetUSDGAmount(token string) *big.Int {
	supply := v.USDG.TotalSupply

	if supply.Sign() == 0 {
		return bignumber.ZeroBI
	}

	weight := v.TokenWeights[token]

	target := new(big.Int).Mul(weight, supply)
	return target.Div(target, v.TotalTokenWeights)
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
		v.USDGAmounts[token] = bignumber.ZeroBI
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
