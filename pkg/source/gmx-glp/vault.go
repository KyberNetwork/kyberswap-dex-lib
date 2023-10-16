package gmxglp

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/ethereum/go-ethereum/common"
)

type Vault struct {
	HasDynamicFees           bool     `json:"hasDynamicFees"`
	IncludeAmmPrice          bool     `json:"includeAmmPrice"`
	IsSwapEnabled            bool     `json:"isSwapEnabled"`
	StableSwapFeeBasisPoints *big.Int `json:"stableSwapFeeBasisPoints"`
	StableTaxBasisPoints     *big.Int `json:"stableTaxBasisPoints"`
	SwapFeeBasisPoints       *big.Int `json:"swapFeeBasisPoints"`
	TotalTokenWeights        *big.Int `json:"totalTokenWeights"`
	TaxBasisPoints           *big.Int `json:"taxBasisPoints"`
	MintBurnFeeBasicPoints   *big.Int `json:"mintBurnFeeBasicPoints"`

	WhitelistedTokens []string            `json:"whitelistedTokens"`
	PoolAmounts       map[string]*big.Int `json:"poolAmounts"`
	BufferAmounts     map[string]*big.Int `json:"bufferAmounts"`
	ReservedAmounts   map[string]*big.Int `json:"reservedAmounts"`
	TokenDecimals     map[string]*big.Int `json:"tokenDecimals"`
	StableTokens      map[string]bool     `json:"stableTokens"`
	USDGAmounts       map[string]*big.Int `json:"usdgAmounts"`
	MaxUSDGAmounts    map[string]*big.Int `json:"maxUsdgAmounts"`
	TokenWeights      map[string]*big.Int `json:"tokenWeights"`

	PriceFeedAddress common.Address  `json:"-"`
	PriceFeed        *VaultPriceFeed `json:"priceFeed"`

	USDGAddress common.Address `json:"-"`
	USDG        *USDG          `json:"usdg"`

	WhitelistedTokensCount *big.Int `json:"-"`

	UseSwapPricing bool // not used, always false for now
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
	vaultMethodUSDG                     = "usdg"
	vaultMethodWhitelistedTokenCount    = "whitelistedTokenCount"
	vaultMethodMintBurnFeeBasisPoints   = "mintBurnFeeBasisPoints"

	vaultMethodAllWhitelistedTokens = "allWhitelistedTokens"

	vaultMethodPoolAmounts     = "poolAmounts"
	vaultMethodBufferAmounts   = "bufferAmounts"
	vaultMethodReservedAmounts = "reservedAmounts"
	vaultMethodTokenDecimals   = "tokenDecimals"
	vaultMethodStableTokens    = "stableTokens"
	vaultMethodUSDGAmounts     = "usdgAmounts"
	vaultMethodMaxUSDGAmounts  = "maxUsdgAmounts"
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

	if supply.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
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

func (v *Vault) CollectSwapFees(tokenIn string, amount, feeBasisPoints *big.Int) (*big.Int, error) {
	afterFeeAmount, err := sub(BasisPointsDivisor, feeBasisPoints)
	if err != nil {
		return nil, err
	}
	afterFeeAmount, err = mul(amount, afterFeeAmount)
	if err != nil {
		return nil, err
	}
	afterFeeAmount, err = div(afterFeeAmount, BasisPointsDivisor)
	if err != nil {
		return nil, err
	}

	return afterFeeAmount, nil
}
