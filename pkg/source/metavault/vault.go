package metavault

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type Vault struct {
	BlockNumber              *big.Int `json:"-"`
	HasDynamicFees           bool     `json:"hasDynamicFees"`
	IncludeAmmPrice          bool     `json:"includeAmmPrice"`
	IsSwapEnabled            bool     `json:"isSwapEnabled"`
	StableSwapFeeBasisPoints *big.Int `json:"stableSwapFeeBasisPoints"`
	StableTaxBasisPoints     *big.Int `json:"stableTaxBasisPoints"`
	SwapFeeBasisPoints       *big.Int `json:"swapFeeBasisPoints"`
	TaxBasisPoints           *big.Int `json:"taxBasisPoints"`
	TotalTokenWeights        *big.Int `json:"totalTokenWeights"`

	WhitelistedTokens []string            `json:"whitelistedTokens"`
	PoolAmounts       map[string]*big.Int `json:"poolAmounts"`
	BufferAmounts     map[string]*big.Int `json:"bufferAmounts"`
	ReservedAmounts   map[string]*big.Int `json:"reservedAmounts"`
	TokenDecimals     map[string]*big.Int `json:"tokenDecimals"`
	StableTokens      map[string]bool     `json:"stableTokens"`
	USDMAmounts       map[string]*big.Int `json:"usdmAmounts"`
	MaxUSDMAmounts    map[string]*big.Int `json:"maxUsdmAmounts"`
	TokenWeights      map[string]*big.Int `json:"tokenWeights"`

	// TokenBalances is the vault accounting of each token's balance (Vault.tokenBalances) and RealBalances is
	// the ERC20 balanceOf(vault), both read at the same block. A nil entry (or an extra saved before these
	// fields existed) means unknown and is never treated as drained.
	TokenBalances map[string]*big.Int `json:"tokenBalances,omitempty"`
	RealBalances  map[string]*big.Int `json:"realBalances,omitempty"`

	PriceFeedAddress common.Address  `json:"-"`
	PriceFeed        *VaultPriceFeed `json:"priceFeed"`

	USDMAddress common.Address `json:"-"`
	USDM        *USDM          `json:"usdm"`

	WhitelistedTokensCount *big.Int `json:"-"`

	UseSwapPricing bool // not used for now, always false
}

func NewVault() *Vault {
	return &Vault{
		PoolAmounts:     make(map[string]*big.Int),
		BufferAmounts:   make(map[string]*big.Int),
		ReservedAmounts: make(map[string]*big.Int),
		TokenDecimals:   make(map[string]*big.Int),
		StableTokens:    make(map[string]bool),
		USDMAmounts:     make(map[string]*big.Int),
		MaxUSDMAmounts:  make(map[string]*big.Int),
		TokenWeights:    make(map[string]*big.Int),
		TokenBalances:   make(map[string]*big.Int),
		RealBalances:    make(map[string]*big.Int),
	}
}

const (
	VaultMethodHasDynamicFees           = "hasDynamicFees"
	VaultMethodIsSwapEnabled            = "isSwapEnabled"
	VaultMethodPriceFeed                = "priceFeed"
	VaultMethodStableSwapFeeBasisPoints = "stableSwapFeeBasisPoints"
	VaultMethodStableTaxBasisPoints     = "stableTaxBasisPoints"
	VaultMethodSwapFeeBasisPoints       = "swapFeeBasisPoints"
	VaultMethodTaxBasisPoints           = "taxBasisPoints"
	VaultMethodTotalTokenWeights        = "totalTokenWeights"
	VaultMethodUSDM                     = "usdm"
	VaultMethodWhitelistedTokenCount    = "whitelistedTokenCount"

	VaultMethodAllWhitelistedTokens = "allWhitelistedTokens"

	VaultMethodPoolAmounts     = "poolAmounts"
	VaultMethodBufferAmounts   = "bufferAmounts"
	VaultMethodReservedAmounts = "reservedAmounts"
	VaultMethodTokenDecimals   = "tokenDecimals"
	VaultMethodStableTokens    = "stableTokens"
	VaultMethodUSDMAmounts     = "usdmAmounts"
	VaultMethodMaxUSDMAmounts  = "maxUsdmAmounts"
	VaultMethodTokenWeights    = "tokenWeights"
	VaultMethodTokenBalances   = "tokenBalances"

	ERC20MethodBalanceOf = "balanceOf"
)

func (v *Vault) GetMinPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, false, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetMaxPrice(token string) (*big.Int, error) {
	return v.PriceFeed.GetPrice(token, true, v.IncludeAmmPrice, v.UseSwapPricing)
}

func (v *Vault) GetTargetUSDMAmount(token string) *big.Int {
	supply := v.USDM.TotalSupply

	if supply.Cmp(bignumber.ZeroBI) == 0 {
		return bignumber.ZeroBI
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
		v.USDMAmounts[token] = bignumber.ZeroBI
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

// CreditedAmountIn mirrors Vault._transferIn (called by swap with the amount the router sent to the vault already
// in its balance):
//
//	nextBalance = balanceOf(vault); amountIn = nextBalance.sub(tokenBalances[token]); tokenBalances[token] = nextBalance
//
// so the vault credits the user with the sent amount plus the surplus of the real balance over its accounting
// (a donation), or minus the deficit. A deficit above the sent amount underflows (SafeMath) and a credit of 0 fails
// swap's `amountIn > 0` check (code 27). A real balance or an accounting that is unknown (nil, e.g. an extra saved
// before these fields existed) is not modelled: the sent amount is credited.
func (v *Vault) CreditedAmountIn(token string, sent *big.Int) (*big.Int, error) {
	realBalance, tokenBalance := v.RealBalances[token], v.TokenBalances[token]
	if realBalance == nil || tokenBalance == nil {
		return sent, nil
	}
	credited := new(big.Int).Sub(new(big.Int).Add(realBalance, sent), tokenBalance)
	if credited.Sign() < 0 {
		return nil, ErrVaultSubOverflow
	}
	if credited.Sign() == 0 {
		return nil, ErrVaultInvalidAmountIn
	}
	return credited, nil
}

// SyncBalancesAfterSwap mirrors the balance part of swap on the real balances: _transferIn leaves
// tokenBalances[in] = balanceOf(in) = realIn + sent and _transferOut leaves tokenBalances[out] = balanceOf(out) =
// realOut - amountOutAfterFees. A real balance that is unknown (nil) stays unknown, and so does an unknown accounting.
func (v *Vault) SyncBalancesAfterSwap(tokenIn string, sent *big.Int, tokenOut string, amountOutAfterFees *big.Int) {
	v.moveRealBalance(tokenIn, sent)
	v.moveRealBalance(tokenOut, new(big.Int).Neg(amountOutAfterFees))
}

func (v *Vault) moveRealBalance(token string, delta *big.Int) {
	realBalance := v.RealBalances[token]
	if realBalance == nil {
		return
	}
	newBalance := new(big.Int).Add(realBalance, delta)
	v.RealBalances[token] = newBalance
	if v.TokenBalances[token] != nil {
		v.TokenBalances[token] = newBalance
	}
}
