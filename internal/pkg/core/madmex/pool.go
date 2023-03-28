package madmex

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

type Gas struct {
	Swap int64
}

type Pool struct {
	pool.Pool

	vault      *Vault
	vaultUtils *VaultUtils
	gas        Gas
}

func NewPool(entityPool entity.Pool) (*Pool, error) {
	var extra Extra
	if err := json.Unmarshal([]byte(entityPool.Extra), &extra); err != nil {
		return nil, err
	}

	tokens := make([]string, 0, len(entityPool.Tokens))
	for _, poolToken := range entityPool.Tokens {
		tokens = append(tokens, poolToken.Address)
	}

	info := pool.PoolInfo{
		Address:  entityPool.Address,
		Exchange: entityPool.Exchange,
		Type:     entityPool.Type,
		Tokens:   tokens,
	}

	return &Pool{
		Pool: pool.Pool{
			Info: info,
		},
		vault:      extra.Vault,
		vaultUtils: NewVaultUtils(extra.Vault),
		gas:        DefaultGas,
	}, nil
}

func (p *Pool) CalcAmountOut(
	tokenAmountIn pool.TokenAmount,
	tokenOut string,
) (*pool.CalcAmountOutResult, error) {
	amountOutAfterFees, feeAmount, err := p.getAmountOut(tokenAmountIn.Token, tokenOut, tokenAmountIn.Amount)
	if err != nil {
		return &pool.CalcAmountOutResult{}, err
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
	}, nil
}

// UpdateBalance update UsdgAmount only
// https://github.com/gmx-io/gmx-contracts/blob/787d767e033c411f6d083f2725fb54b7fa956f7e/contracts/core/Vault.sol#L547-L548
func (p *Pool) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output, fee := params.TokenAmountIn, params.TokenAmountOut, params.Fee
	priceIn, err := p.vault.GetMinPrice(input.Token)
	if err != nil {
		return
	}

	usdgAmount := new(big.Int).Div(new(big.Int).Mul(input.Amount, priceIn), PricePrecision)
	usdgAmount = p.vault.AdjustForDecimals(usdgAmount, input.Token, p.vault.USDG.Address)

	p.vault.IncreaseUSDGAmount(input.Token, usdgAmount)
	p.vault.DecreaseUSDGAmount(output.Token, usdgAmount)
	p.vault.IncreasePoolAmount(input.Token, input.Amount)
	p.vault.DecreasePoolAmount(output.Token, new(big.Int).Add(output.Amount, fee.Amount))
}

func (p *Pool) CanSwapTo(address string) []string {
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

func (p *Pool) GetLpToken() string { return "" }

func (p *Pool) GetMidPrice(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	amountOutAfterFees, feeAmount, err := p.getAmountOut(tokenIn, tokenOut, base)
	if err != nil {
		return nil
	}

	return new(big.Int).Add(amountOutAfterFees, feeAmount)
}

func (p *Pool) CalcExactQuote(tokenIn string, tokenOut string, base *big.Int) *big.Int {
	amountOutAfterFees, feeAmount, err := p.getAmountOut(tokenIn, tokenOut, base)
	if err != nil {
		return constant.Zero
	}

	return new(big.Int).Add(amountOutAfterFees, feeAmount)
}

func (p *Pool) GetMetaInfo(_ string, _ string) interface{} { return nil }

// getAmountOut returns amountOutAfterFees, feeAmount and error
func (p *Pool) getAmountOut(tokenIn string, tokenOut string, amountIn *big.Int) (*big.Int, *big.Int, error) {
	if !p.vault.IsSwapEnabled {
		return nil, nil, ErrVaultSwapsNotEnabled
	}

	priceIn, err := p.vault.GetMinPrice(tokenIn)
	if err != nil {
		return nil, nil, err
	}

	priceOut, err := p.vault.GetMaxPrice(tokenOut)
	if err != nil {
		return nil, nil, err
	}

	amountOut := new(big.Int).Div(new(big.Int).Mul(amountIn, priceIn), priceOut)
	amountOut = p.vault.AdjustForDecimals(amountOut, tokenIn, tokenOut)

	usdgAmount := new(big.Int).Div(new(big.Int).Mul(amountIn, priceIn), PricePrecision)
	usdgAmount = p.vault.AdjustForDecimals(usdgAmount, tokenIn, p.vault.USDG.Address)

	// in smart contract, this validation is implemented inside _increaseUsdgAmount method
	if err = p.validateMaxUsdgExceed(tokenIn, usdgAmount); err != nil {
		return nil, nil, err
	}

	// in smart contract, this validation is implemented inside _decreasePoolAmount method
	if err = p.validateMinPoolAmount(tokenOut, amountOut); err != nil {
		return nil, nil, err
	}

	// in smart contract, this validation is implemented inside _validateBufferAmount method
	if err = p.validateBufferAmount(tokenOut, amountOut); err != nil {
		return nil, nil, err
	}

	feeBasisPoints := p.vaultUtils.GetSwapFeeBasisPoints(tokenIn, tokenOut, usdgAmount)
	amountOutAfterFees := new(big.Int).Div(
		new(big.Int).Mul(
			amountOut,
			new(big.Int).Sub(BasisPointsDivisor, feeBasisPoints),
		),
		BasisPointsDivisor,
	)

	feeAmount := new(big.Int).Sub(amountOut, amountOutAfterFees)

	return amountOutAfterFees, feeAmount, nil
}

func (p *Pool) validateMaxUsdgExceed(token string, amount *big.Int) error {
	currentUsdgAmount := p.vault.USDGAmounts[token]
	newUsdgAmount := new(big.Int).Add(currentUsdgAmount, amount)

	maxUsdgAmount := p.vault.MaxUSDGAmounts[token]

	if maxUsdgAmount.Cmp(constant.Zero) == 0 {
		return nil
	}

	if newUsdgAmount.Cmp(maxUsdgAmount) < 0 {
		return nil
	}

	return ErrVaultMaxUsdgExceeded
}

func (p *Pool) validateMinPoolAmount(token string, amount *big.Int) error {
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

func (p *Pool) validateBufferAmount(token string, amount *big.Int) error {
	currentPoolAmount := p.vault.PoolAmounts[token]
	newPoolAmount := new(big.Int).Sub(currentPoolAmount, amount)

	bufferAmount := p.vault.BufferAmounts[token]

	if newPoolAmount.Cmp(bufferAmount) < 0 {
		return ErrVaultPoolAmountLessThanBufferAmount
	}

	return nil
}
