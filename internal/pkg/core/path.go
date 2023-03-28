package core

import (
	"math/big"

	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	poolPkg "github.com/KyberNetwork/router-service/internal/pkg/core/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

var (
	ErrInvalidTokenLength = errors.New("invalid token length, token length should be more than 1")
	ErrInvalidPoolLength  = errors.New("invalid pool length, pool length should be less than token length 1")
	ErrInvalidTokenIn     = errors.New("invalid tokenIn, the first token does not match tokenIn")
	ErrInvalidTokenOut    = errors.New("invalid tokenOut, the last token does not match tokenOut")
)

type Path struct {
	// Input consists of tokenIn and amountIn
	Input poolPkg.TokenAmount

	// Output consists of tokenOut and amountOut
	Output poolPkg.TokenAmount

	// TotalGas estimated gas required swapping through this path
	TotalGas int64

	// Pools list pools that path swap through, length of pools = length of tokens - 1
	Pools []poolPkg.IPool

	// Tokens list tokens that path swap through
	Tokens []entity.Token

	// PriceImpact (1 - exactQuote/amountOut) in 18 decimals
	PriceImpact *big.Int
}

func NewPath(
	pools []poolPkg.IPool,
	tokens []entity.Token,
	tokenAmountIn poolPkg.TokenAmount,
	tokenOut string,
	tokenOutPrice float64,
	tokenOutDecimals uint8,
	gasOption GasOption,
) (*Path, error) {
	var (
		tokenLen = len(tokens)
		poolLen  = len(pools)
	)

	if tokenLen < 2 {
		return nil, ErrInvalidTokenLength
	}

	if poolLen+1 != tokenLen {
		return nil, ErrInvalidPoolLength
	}

	if tokens[0].Address != tokenAmountIn.Token {
		return nil, ErrInvalidTokenIn
	}

	if tokens[tokenLen-1].Address != tokenOut {
		return nil, ErrInvalidTokenOut
	}

	path := Path{
		Input:  tokenAmountIn,
		Pools:  pools,
		Tokens: tokens,
	}

	tokenAmountOut, totalGas, err := path.calcAmountOut()
	if err != nil {
		return nil, err
	}

	amountUSD := utils.CalcTokenAmountUsd(tokenAmountOut.Amount, tokenOutDecimals, tokenOutPrice)
	totalGasUSD := utils.CalcGasUsd(gasOption.Price, totalGas, gasOption.TokenPrice)
	tokenAmountOut.AmountUsd = amountUSD - totalGasUSD

	path.Output = tokenAmountOut
	path.TotalGas = totalGas
	path.PriceImpact = path.calcPriceImpact()

	return &path, nil
}

// TrySwap tries swap through path with tokenAmountIn and return tokenAmountOut
func (p *Path) TrySwap(tokenAmountIn poolPkg.TokenAmount) (poolPkg.TokenAmount, error) {
	tokenAmountOut := tokenAmountIn

	for i, pool := range p.Pools {
		calcAmountOutResult, err := pool.CalcAmountOut(tokenAmountOut, p.Tokens[i+1].Address)
		if err != nil {
			return poolPkg.TokenAmount{}, errors.Wrapf(
				ErrInvalidSwap,
				"[Path.calcAmountOut] calcAmountOut returns error | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s], err: [%v]",
				pool.GetAddress(),
				pool.GetExchange(),
				tokenAmountOut.Token,
				tokenAmountOut.Amount,
				p.Tokens[i+1].Address,
				err,
			)
		}
		swapTokenAmountOut := calcAmountOutResult.TokenAmountOut
		if swapTokenAmountOut == nil {
			return poolPkg.TokenAmount{}, errors.Wrapf(
				ErrInvalidSwap,
				"[Path.calcAmountOut] calcAmountOut returns nil | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s]",
				pool.GetAddress(),
				pool.GetExchange(),
				tokenAmountOut.Token,
				tokenAmountOut.Amount,
				p.Tokens[i+1].Address,
			)
		}

		tokenAmountOut = *swapTokenAmountOut
	}

	return tokenAmountOut, nil
}

// Equals returns true when two paths have same token and pool in respective order
func (p *Path) Equals(other *Path) bool {
	if len(p.Pools) != len(other.Pools) || len(p.Tokens) != len(other.Tokens) {
		return false
	}

	if p.Input.Token != other.Input.Token || p.Output.Token != other.Output.Token {
		return false
	}

	for idx := range p.Pools {
		if p.Tokens[idx] != other.Tokens[idx] || !p.Pools[idx].Equals(other.Pools[idx]) {
			return false
		}
	}

	return true
}

// Merge merges other path if two paths are equal
// - Input: add up other path Amount and AmountUsd
// - Output: add up other path Amount and AmountUsd
func (p *Path) Merge(other *Path) bool {
	if !p.Equals(other) {
		return false
	}

	newInput := poolPkg.TokenAmount{
		Token:     p.Input.Token,
		Amount:    new(big.Int).Add(p.Input.Amount, other.Input.Amount),
		AmountUsd: p.Input.AmountUsd + other.Input.AmountUsd,
	}

	newOutput := poolPkg.TokenAmount{
		Token:     p.Output.Token,
		Amount:    new(big.Int).Add(p.Output.Amount, other.Output.Amount),
		AmountUsd: p.Output.AmountUsd + other.Output.AmountUsd,
	}

	p.Input = newInput
	p.Output = newOutput

	return true
}

// CompareTo compares with other path and returns
//
//	-1: if other path is nil or worse than current path
//	1: if other path is better than current path
//	0: if other path is same as current path
func (p *Path) CompareTo(other *Path, gasInclude bool) int {
	if other == nil {
		return -1
	}

	if amountCmp := p.cmpAmounts(other, gasInclude); amountCmp != 0 {
		return amountCmp
	}

	if priceImpactCmp := p.PriceImpact.Cmp(other.PriceImpact); priceImpactCmp != 0 {
		return priceImpactCmp
	}

	return p.cmpTokenLen(other)
}

// calcAmountOut swaps through path with Input
func (p *Path) calcAmountOut() (poolPkg.TokenAmount, int64, error) {
	tokenAmountOut := p.Input
	var totalGas int64

	for i, pool := range p.Pools {
		calcAmountOutResult, err := pool.CalcAmountOut(tokenAmountOut, p.Tokens[i+1].Address)
		if err != nil {
			return poolPkg.TokenAmount{}, 0, errors.Wrapf(
				ErrInvalidSwap,
				"[Path.calcAmountOut] calcAmountOut returns error | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s], err: [%v]",
				pool.GetAddress(),
				pool.GetExchange(),
				tokenAmountOut.Token,
				tokenAmountOut.Amount,
				p.Tokens[i+1].Address,
				err,
			)
		}
		swapTokenAmountOut, gas := calcAmountOutResult.TokenAmountOut, calcAmountOutResult.Gas
		if swapTokenAmountOut == nil {
			return poolPkg.TokenAmount{}, 0, errors.Wrapf(
				ErrInvalidSwap,
				"[Path.calcAmountOut] calcAmountOut returns nil | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s]",
				pool.GetAddress(),
				pool.GetExchange(),
				tokenAmountOut.Token,
				tokenAmountOut.Amount,
				p.Tokens[i+1].Address,
			)
		}

		tokenAmountOut = *swapTokenAmountOut
		totalGas += gas
	}

	return tokenAmountOut, totalGas, nil
}

func (p *Path) calcPriceImpact() *big.Int {
	exactQuote := p.calcExactQuote()

	// Return price impact = 100% (10^18 * 100) when exactQuote <= 0
	if exactQuote.Cmp(big.NewInt(0)) < 1 {
		return new(big.Int).Mul(constant.TenPowInt(18), big.NewInt(100))
	}

	// 10^18 * (1 - amountOut/exactQuote)
	return new(big.Int).Sub(
		constant.BONE,
		new(big.Int).Div(
			new(big.Int).Mul(constant.BONE, p.Output.Amount),
			exactQuote,
		),
	)
}

func (p *Path) calcExactQuote() *big.Int {
	exactQuote := new(big.Int).Mul(p.Input.Amount, constant.BONE)
	for i, pool := range p.Pools {
		exactQuote = pool.CalcExactQuote(p.Tokens[i].Address, p.Tokens[i+1].Address, exactQuote)
	}

	return new(big.Int).Div(exactQuote, constant.BONE)
}

func (p *Path) cmpAmounts(other *Path, gasInclude bool) int {
	if gasInclude {
		if amountUSDCmp := p.cmpAmountUSD(other); amountUSDCmp != 0 {
			return amountUSDCmp
		}
	}

	return p.cmpAmount(other)
}

// cmpAmount compares p and other and returns
//
//	-1 if:
//		- p.Output.Amount > other.Output.Amount (or)
//	  	- p.Output.Amount == other.Output.Amount && p.Input.Amount < other.Input.Amount
//	+1 if:
//		- p.Output.Amount > other.Output.Amount (or)
//	  	- p.Output.Amount == other.Output.Amount && p.Input.Amount > other.Input.Amount
//	0 if:
//	  	- p.Output.Amount = other.Output.Amount && p.Input.Amount = other.Input.Amount
func (p *Path) cmpAmount(other *Path) int {
	outputCmp := p.Output.Amount.Cmp(other.Output.Amount)

	if outputCmp != 0 {
		return -outputCmp
	}

	return p.Input.Amount.Cmp(other.Input.Amount)
}

// cmpAmountUSD compares p and other and returns
//
//	-1 if p.output.AmountUSD > other.Output.AmountUsd
//	1 if p.output.AmountUSD > other.Output.AmountUsd
//	0 if p.output.AmountUSD = other.Output.AmountUsd
func (p *Path) cmpAmountUSD(other *Path) int {
	if p.Output.AmountUsd > other.Output.AmountUsd {
		return -1
	}

	if p.Output.AmountUsd < other.Output.AmountUsd {
		return 1
	}

	return 0
}

// cmpTokenLen compares p and other and returns
//
//	-1 if len(p.Tokens) > len(other.Tokens)
//	1 if len(p.Tokens) < len(other.Tokens)
//	0 if len(p.Tokens) = len(other.Tokens)
func (p *Path) cmpTokenLen(other *Path) int {
	tokenLenDiff := len(p.Tokens) - len(other.Tokens)

	if tokenLenDiff > 0 {
		return 1
	}

	if tokenLenDiff < 0 {
		return -1
	}

	return 0
}
