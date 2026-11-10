package metavault

import (
	"maps"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type Gas struct {
	Swap int64
}

type PoolSimulator struct {
	pool.Pool

	vault      *Vault
	vaultUtils *VaultUtils
	gas        Gas
}

var _ = pool.RegisterFactory0(DexTypeMetavault, NewPoolSimulator)

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, 0, len(entityPool.Tokens))
	for _, poolToken := range entityPool.Tokens {
		tokens = append(tokens, poolToken.Address)
	}

	reserves := make([]*big.Int, len(entityPool.Reserves))
	for i, reserve := range entityPool.Reserves {
		reserveBI, ok := new(big.Int).SetString(reserve, 10)
		if !ok {
			reserveBI = integer.Zero()
		}

		reserves[i] = reserveBI
	}

	info := pool.PoolInfo{
		Address:     entityPool.Address,
		Exchange:    entityPool.Exchange,
		Type:        entityPool.Type,
		Tokens:      tokens,
		Reserves:    reserves,
		BlockNumber: entityPool.BlockNumber,
	}

	return &PoolSimulator{
		Info:       info,
		vault:      extra.Vault,
		vaultUtils: NewVaultUtils(extra.Vault),
		gas:        DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(param pool.CalcAmountOutParams) (*pool.CalcAmountOutResult, error) {
	tokenAmountIn := param.TokenAmountIn
	tokenOut := param.TokenOut
	amountOutAfterFees, feeAmount, creditedIn, usdmAmount, err := p.getAmountOut(tokenAmountIn.Token, tokenOut,
		tokenAmountIn.Amount)
	if err != nil {
		return nil, err
	}

	tokenAmountOut := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: amountOutAfterFees,
	}
	tokenAmountFee := &pool.TokenAmount{
		Token:  tokenOut,
		Amount: feeAmount,
	}

	return &pool.CalcAmountOutResult{
		TokenAmountOut: tokenAmountOut,
		Fee:            tokenAmountFee,
		Gas:            p.gas.Swap,
		SwapInfo:       newSwapInfo(creditedIn, usdmAmount),
	}, nil
}

func (p *PoolSimulator) CloneState() pool.IPoolSimulator {
	cloned := *p
	vault := *p.vault
	vault.USDMAmounts = maps.Clone(p.vault.USDMAmounts)
	vault.PoolAmounts = maps.Clone(p.vault.PoolAmounts)
	vault.TokenBalances = maps.Clone(p.vault.TokenBalances)
	vault.RealBalances = maps.Clone(p.vault.RealBalances)
	cloned.vault = &vault
	return &cloned
}

// UpdateBalance mirrors the state swap leaves behind:
// https://github.com/gmx-io/gmx-contracts/blob/787d767e033c411f6d083f2725fb54b7fa956f7e/contracts/core/Vault.sol#L547-L548
// usdmAmounts and poolAmounts move by the credited amount in (not the sent one) and, when the real balances are
// known, tokenBalances and the real balances follow the transfers (_transferIn / _transferOut). The credited amount
// and the USDM amount come from the SwapInfo of CalcAmountOut, computed from the state before this update, as the
// quote was; without a usable SwapInfo they are computed from the current state, which is the same state when the
// pool was not updated in between. feeReserves is not tracked: no swap pricing or validation reads it.
func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output, fee := params.TokenAmountIn, params.TokenAmountOut, params.Fee
	creditedIn, usdmAmount, ok := p.swapAmountsFrom(params.SwapInfo)
	if !ok {
		var err error
		if creditedIn, _, usdmAmount, err = p.swapAmounts(input.Token, input.Amount); err != nil {
			return
		}
	}

	p.vault.IncreaseUSDMAmount(input.Token, usdmAmount)
	p.vault.DecreaseUSDMAmount(output.Token, usdmAmount)
	p.vault.IncreasePoolAmount(input.Token, creditedIn)
	p.vault.DecreasePoolAmount(output.Token, new(big.Int).Add(output.Amount, fee.Amount))
	p.vault.SyncBalancesAfterSwap(input.Token, input.Amount, output.Token, output.Amount)
}

// swapAmountsFrom reads the amounts from the SwapInfo of CalcAmountOut. ok is false when swapInfo is missing or is not
// the SwapInfo of this pool. The returned values are new, so the state does not share them with the SwapInfo.
func (p *PoolSimulator) swapAmountsFrom(swapInfo any) (creditedIn, usdmAmount *big.Int, ok bool) {
	info, ok := swapInfo.(*SwapInfo)
	if !ok || info == nil {
		return nil, nil, false
	}
	return info.CreditedIn.ToBig(), info.UsdmAmount.ToBig(), true
}

// swapAmounts computes what swap does first: the amount credited for the amount sent, the min price of the token in
// and the value of the credited amount in USDM at that price.
func (p *PoolSimulator) swapAmounts(tokenIn string, sent *big.Int) (creditedIn, priceIn, usdmAmount *big.Int, err error) {
	// in smart contract: amountIn = _transferIn(tokenIn); require(amountIn > 0, 27)
	if creditedIn, err = p.vault.CreditedAmountIn(tokenIn, sent); err != nil {
		return nil, nil, nil, err
	}
	if priceIn, err = p.vault.GetMinPrice(tokenIn); err != nil {
		return nil, nil, nil, err
	}
	usdmAmount = new(big.Int).Div(new(big.Int).Mul(creditedIn, priceIn), PricePrecision)
	usdmAmount = p.vault.AdjustForDecimals(usdmAmount, tokenIn, p.vault.USDM.Address)
	return creditedIn, priceIn, usdmAmount, nil
}

// newSwapInfo returns a nil interface when an amount does not fit 256 bits: UpdateBalance then computes the amounts
// itself.
func newSwapInfo(creditedIn, usdmAmount *big.Int) any {
	var info SwapInfo
	if info.CreditedIn.SetFromBig(creditedIn) {
		return nil
	}
	if info.UsdmAmount.SetFromBig(usdmAmount) {
		return nil
	}
	return &info
}

func (p *PoolSimulator) CanSwapFrom(address string) []string {
	return p.CanSwapTo(address)
}

func (p *PoolSimulator) CanSwapTo(address string) []string {
	whitelistedTokens := p.vault.WhitelistedTokens

	isTokenExists := false
	for _, token := range whitelistedTokens {
		if strings.EqualFold(token, address) {
			isTokenExists = true
		}
	}

	if !isTokenExists {
		return nil
	}

	swappableTokens := make([]string, 0, len(whitelistedTokens)-1)
	for _, token := range whitelistedTokens {
		tokenAddress := token

		if address == tokenAddress {
			continue
		}

		swappableTokens = append(swappableTokens, tokenAddress)
	}

	return swappableTokens
}

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) any {
	return pool.MetaInfo{BlockNumber: p.Info.BlockNumber}
}

// getAmountOut returns amountOutAfterFees, feeAmount, the credited amount in, its USDM amount and error. It follows
// Vault.swap step by step; amountIn is the amount the router sent to the vault, and the vault works with the credited
// amount (see Vault.CreditedAmountIn).
func (p *PoolSimulator) getAmountOut(tokenIn string, tokenOut string, amountIn *big.Int) (
	*big.Int, *big.Int, *big.Int, *big.Int, error) {
	if !p.vault.IsSwapEnabled {
		return nil, nil, nil, nil, ErrVaultSwapsNotEnabled
	}

	creditedIn, priceIn, usdmAmount, err := p.swapAmounts(tokenIn, amountIn)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	priceOut, err := p.vault.GetMaxPrice(tokenOut)
	if err != nil {
		return nil, nil, nil, nil, err
	}

	amountOut := new(big.Int).Div(new(big.Int).Mul(creditedIn, priceIn), priceOut)
	amountOut = p.vault.AdjustForDecimals(amountOut, tokenIn, tokenOut)

	feeBasisPoints := p.vaultUtils.GetSwapFeeBasisPoints(tokenIn, tokenOut, usdmAmount)
	amountOutAfterFees := new(big.Int).Div(
		new(big.Int).Mul(
			amountOut,
			new(big.Int).Sub(BasisPointsDivisor, feeBasisPoints),
		),
		BasisPointsDivisor,
	)

	if err = p.validateMaxUsdmExceed(tokenIn, usdmAmount); err != nil {
		return nil, nil, nil, nil, err
	}

	// in smart contract, this validation is implemented inside _increasePoolAmount method (code 49)
	if err = p.validatePoolAmountWithinBalance(tokenIn, creditedIn, amountIn); err != nil {
		return nil, nil, nil, nil, err
	}

	// in smart contract, this validation is implemented inside _decreasePoolAmount method
	if err = p.validateMinPoolAmount(tokenOut, amountOut); err != nil {
		return nil, nil, nil, nil, err
	}

	// in smart contract, this validation is implemented inside _validateBufferAmount method
	if err = p.validateBufferAmount(tokenOut, amountOut); err != nil {
		return nil, nil, nil, nil, err
	}

	// in smart contract, this is the ERC20 transfer out of amountOutAfterFees (_transferOut), the fee stays in the vault
	if realOut := p.vault.RealBalances[tokenOut]; realOut != nil && realOut.Cmp(amountOutAfterFees) < 0 {
		return nil, nil, nil, nil, ErrInsufficientRealBalance
	}

	feeAmount := new(big.Int).Sub(amountOut, amountOutAfterFees)

	return amountOutAfterFees, feeAmount, creditedIn, usdmAmount, nil
}

// validatePoolAmountWithinBalance mirrors _increasePoolAmount: poolAmounts[token] + credited <= balanceOf(vault),
// where balanceOf(vault) is the real balance before the swap plus the amount sent. Skipped when the real balance is
// unknown.
func (p *PoolSimulator) validatePoolAmountWithinBalance(token string, credited, sent *big.Int) error {
	realBalance := p.vault.RealBalances[token]
	if realBalance == nil {
		return nil
	}
	newPoolAmount := new(big.Int).Add(p.vault.PoolAmounts[token], credited)
	if newPoolAmount.Cmp(new(big.Int).Add(realBalance, sent)) > 0 {
		return ErrVaultPoolAmountExceedsBalance
	}
	return nil
}

func (p *PoolSimulator) validateMaxUsdmExceed(token string, amount *big.Int) error {
	currentUsdmAmount := p.vault.USDMAmounts[token]
	newUsdmAmount := new(big.Int).Add(currentUsdmAmount, amount)

	maxUsdmAmount := p.vault.MaxUSDMAmounts[token]

	if maxUsdmAmount.Cmp(bignumber.ZeroBI) == 0 {
		return nil
	}

	// _increaseUsdmAmount: _validate(usdmAmounts[_token] <= maxUsdmAmount, 51), so reaching the cap is allowed.
	if newUsdmAmount.Cmp(maxUsdmAmount) <= 0 {
		return nil
	}

	return ErrVaultMaxUsdmExceeded
}

func (p *PoolSimulator) validateMinPoolAmount(token string, amount *big.Int) error {
	currentPoolAmount := p.vault.PoolAmounts[token]

	if currentPoolAmount.Cmp(amount) < 0 {
		return ErrVaultPoolAmountExceeded
	}

	newPoolAmount := new(big.Int).Sub(currentPoolAmount, amount)
	reservedAmount := p.vault.ReservedAmounts[token]

	if reservedAmount.Cmp(newPoolAmount) > 0 {
		return ErrVaultReserveExceedsPool
	}

	return nil
}

func (p *PoolSimulator) validateBufferAmount(token string, amount *big.Int) error {
	currentPoolAmount := p.vault.PoolAmounts[token]
	newPoolAmount := new(big.Int).Sub(currentPoolAmount, amount)

	bufferAmount := p.vault.BufferAmounts[token]

	if newPoolAmount.Cmp(bufferAmount) < 0 {
		return ErrVaultPoolAmountLessThanBufferAmount
	}

	return nil
}

func (p *PoolSimulator) AfterMsgpackUnmarshal() error {
	if p.vaultUtils != nil {
		p.vaultUtils.vault = p.vault
	}
	return nil
}
