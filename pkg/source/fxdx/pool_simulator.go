package fxdx

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/blockchain-toolkit/integer"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type Gas struct {
	Swap int64
}

type PoolSimulator struct {
	pool.Pool

	vault    *Vault
	feeUtils *FeeUtilsV2
	gas      Gas
}

func NewPoolSimulator(entityPool entity.Pool) (*PoolSimulator, error) {
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

	vault := extra.Vault
	feeUtils := extra.FeeUtils
	feeUtils.Vault = vault

	return &PoolSimulator{
		Pool: pool.Pool{
			Info: info,
		},
		vault:    vault,
		feeUtils: feeUtils,
		gas:      DefaultGas,
	}, nil
}

func (p *PoolSimulator) CalcAmountOut(
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

func (p *PoolSimulator) UpdateBalance(params pool.UpdateBalanceParams) {
	input, output, fee := params.TokenAmountIn, params.TokenAmountOut, params.Fee
	priceIn, err := p.vault.GetMinPrice(input.Token)
	if err != nil {
		return
	}

	usdfAmount := new(big.Int).Div(new(big.Int).Mul(input.Amount, priceIn), PricePrecision)
	usdfAmount = p.vault.AdjustForDecimals(usdfAmount, input.Token, p.vault.USDF.Address)

	p.vault.IncreaseUSDFAmount(input.Token, usdfAmount)
	p.vault.DecreaseUSDFAmount(output.Token, usdfAmount)
	p.vault.IncreasePoolAmount(input.Token, input.Amount)
	p.vault.DecreasePoolAmount(output.Token, new(big.Int).Add(output.Amount, fee.Amount))
}

func (p *PoolSimulator) CanSwapFrom(address string) []string { return p.CanSwapTo(address) }

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

func (p *PoolSimulator) GetMetaInfo(_ string, _ string) interface{} { return nil }

func (p *PoolSimulator) getAmountOut(tokenIn string, tokenOut string, amountIn *big.Int) (*big.Int, *big.Int, error) {
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

	usdfAmount := new(big.Int).Div(new(big.Int).Mul(amountIn, priceIn), PricePrecision)
	usdfAmount = p.vault.AdjustForDecimals(usdfAmount, tokenIn, p.vault.USDF.Address)

	// in smart contract, this validation is implemented inside _increaseUsdfAmount method
	if err = p.validateMaxUsdfExceed(tokenIn, usdfAmount); err != nil {
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

	feeBasisPoints, err := p.feeUtils.GetSwapFeeBasisPoints(tokenIn, tokenOut, usdfAmount)
	if err != nil {
		return nil, nil, err
	}
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

func (p *PoolSimulator) validateMaxUsdfExceed(token string, amount *big.Int) error {
	currentUsdfAmount := p.vault.USDFAmounts[token]
	newUsdfAmount := new(big.Int).Add(currentUsdfAmount, amount)

	maxUsdfAmount := p.vault.MaxUSDFAmounts[token]

	if maxUsdfAmount.Cmp(integer.Zero()) == 0 {
		return nil
	}

	if newUsdfAmount.Cmp(maxUsdfAmount) < 0 {
		return nil
	}

	return ErrVaultMaxUsdfExceeded
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
