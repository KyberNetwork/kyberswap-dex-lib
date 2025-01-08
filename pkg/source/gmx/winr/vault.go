package winr

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/gmx"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/ethereum/go-ethereum/common"
)

type Vault struct {
	HasDynamicFees           bool     `json:"hasDynamicFees"`
	IsSwapEnabled            bool     `json:"isSwapEnabled"`
	StableSwapFeeBasisPoints *big.Int `json:"stableSwapFeeBasisPoints"`
	StableTaxBasisPoints     *big.Int `json:"stableTaxBasisPoints"`
	SwapFeeBasisPoints       *big.Int `json:"swapFeeBasisPoints"`
	TaxBasisPoints           *big.Int `json:"taxBasisPoints"`
	TotalTokenWeights        *big.Int `json:"totalTokenWeights"`

	WhitelistedTokens []string            `json:"whitelistedTokens"`
	PoolAmounts       map[string]*big.Int `json:"poolAmounts"`
	BufferAmounts     map[string]*big.Int `json:"bufferAmounts"`
	TokenDecimals     map[string]*big.Int `json:"tokenDecimals"`
	StableTokens      map[string]bool     `json:"stableTokens"`
	USDWAmounts       map[string]*big.Int `json:"usdwAmounts"`
	MaxUSDWAmounts    map[string]*big.Int `json:"maxUsdwAmounts"`
	TokenWeights      map[string]*big.Int `json:"tokenWeights"`

	PriceFeedAddress common.Address      `json:"-"`
	PriceFeed        *gmx.VaultPriceFeed `json:"priceFeed"`

	PriceOracleAddress common.Address `json:"-"`

	USDWAddress common.Address `json:"-"`
	USDW        *gmx.USDG      `json:"usdw"`

	WhitelistedTokensCount *big.Int `json:"-"`
}

func NewVault() *Vault {
	return &Vault{
		PoolAmounts:    make(map[string]*big.Int),
		BufferAmounts:  make(map[string]*big.Int),
		TokenDecimals:  make(map[string]*big.Int),
		StableTokens:   make(map[string]bool),
		USDWAmounts:    make(map[string]*big.Int),
		MaxUSDWAmounts: make(map[string]*big.Int),
		TokenWeights:   make(map[string]*big.Int),
	}
}

const (
	vaultMethodPriceOracleRouter = "priceOracleRouter" // [1]
	vaultMethodUSDW              = "usdw"              // [2] usdw
	vaultMethodPrimaryPriceFeed  = "primaryPriceFeed"  // [1]

	// vaultMethodWhitelistedTokens    = "whitelistedTokens" // [3] no need to check (all whitelisted tokens available)

	// vaultMethodReservedAmounts = "reservedAmounts" // [4] -> circuit breaker -> affect isSwapEnabled -> no action since it's kept syncing
	vaultMethodUSDWAmounts    = "usdwAmounts"    // [2] usdw
	vaultMethodMaxUSDWAmounts = "maxUsdwAmounts" // [2] usdw
)

func (v *Vault) GetMinPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, false, false, false)
}

func (v *Vault) GetMaxPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, true, true, false)
}

func (v *Vault) GetTargetUSDWAmount(token string) *big.Int {
	supply := v.USDW.TotalSupply

	if supply.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
	}

	weight := v.TokenWeights[token]

	return new(big.Int).Div(new(big.Int).Mul(weight, supply), v.TotalTokenWeights)
}

func (v *Vault) AdjustForDecimals(amount *big.Int, tokenDiv string, tokenMul string) *big.Int {
	var decimalsDiv *big.Int
	if tokenDiv == v.USDW.Address {
		decimalsDiv = USDWDecimals
	} else {
		decimalsDiv = v.TokenDecimals[tokenDiv]
	}

	var decimalsMul *big.Int
	if tokenMul == v.USDW.Address {
		decimalsMul = USDWDecimals
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
	v.USDWAmounts[token] = new(big.Int).Add(v.USDWAmounts[token], amount)
}

func (v *Vault) DecreaseUSDGAmount(token string, amount *big.Int) {
	currentUSDGAmount := v.USDWAmounts[token]

	if currentUSDGAmount.Cmp(amount) < 0 {
		v.USDWAmounts[token] = bignumber.ZeroBI
		return
	}

	v.USDWAmounts[token] = new(big.Int).Sub(v.USDWAmounts[token], amount)
}

func (v *Vault) IncreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Add(v.PoolAmounts[token], amount)
}

func (v *Vault) DecreasePoolAmount(token string, amount *big.Int) {
	v.PoolAmounts[token] = new(big.Int).Sub(v.PoolAmounts[token], amount)
}
