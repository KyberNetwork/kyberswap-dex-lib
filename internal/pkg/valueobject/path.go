package valueobject

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

var (
	ErrInvalidTokenLength = errors.New("invalid token length, token length should be more than 1")
	ErrInvalidPoolLength  = errors.New("invalid pool length, pool length should be less than token length 1")
	ErrInvalidTokenIn     = errors.New("invalid tokenIn, the first token does not match tokenIn")
	ErrInvalidTokenOut    = errors.New("invalid tokenOut, the last token does not match tokenOut")

	ErrNoIPool     = errors.New("cannot get IPool from address")
	ErrInvalidSwap = errors.New("invalid swap")
)

type Path struct {
	// Input consists of tokenIn and amountIn
	Input poolpkg.TokenAmount `json:"input"`

	// Output consists of tokenOut and amountOut
	Output poolpkg.TokenAmount `json:"output"`

	// TotalGas estimated gas required swapping through this path
	TotalGas int64 `json:"totalGas"`

	// PoolAddresses list address pools that path swap through, length of pools = length of tokens - 1
	PoolAddresses []string `json:"poolAddresses"`

	// Tokens list tokens that path swap through
	Tokens []entity.Token `json:"tokens"`
}

func NewPath(
	poolBucket *PoolBucket,
	poolAddresses []string,
	tokens []entity.Token,
	tokenAmountIn poolpkg.TokenAmount,
	tokenOut string,
	tokenOutPrice float64,
	tokenOutDecimals uint8,
	gasOption GasOption,
) (*Path, error) {
	var (
		tokenLen = len(tokens)
		poolLen  = len(poolAddresses)
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
		Input:         tokenAmountIn,
		PoolAddresses: poolAddresses,
		Tokens:        tokens,
	}

	tokenAmountOut, totalGas, err := path.CalcAmountOut(poolBucket, tokenAmountIn)
	if err != nil {
		return nil, err
	}

	amountUSD := utils.CalcTokenAmountUsd(tokenAmountOut.Amount, tokenOutDecimals, tokenOutPrice)
	totalGasUSD := utils.CalcGasUsd(gasOption.Price, totalGas, gasOption.TokenPrice)
	tokenAmountOut.AmountUsd = amountUSD - totalGasUSD

	path.Output = tokenAmountOut
	path.TotalGas = totalGas

	return &path, nil
}

// CalcAmountOut swaps through path with Input
func (p *Path) CalcAmountOut(poolBucket *PoolBucket, tokenAmountIn poolpkg.TokenAmount) (poolpkg.TokenAmount, int64, error) {
	var (
		currentAmount = tokenAmountIn
		pool          poolpkg.IPoolSimulator
		ok            bool
		totalGas      int64
	)

	for i, poolAddress := range p.PoolAddresses {
		if pool, ok = poolBucket.GetPool(poolAddress); !ok {
			return poolpkg.TokenAmount{}, 0, errors.Wrapf(
				ErrNoIPool,
				"[Path.CalcAmountOut] poolAddress: [%s]",
				poolAddress,
			)
		}
		calcAmountOutResult, err := poolpkg.CalcAmountOut(pool, currentAmount, p.Tokens[i+1].Address)
		if err != nil {
			return poolpkg.TokenAmount{}, 0, errors.Wrapf(
				ErrInvalidSwap,
				"[Path.CalcAmountOut] CalcAmountOut returns error | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s], err: [%v]",
				pool.GetAddress(),
				pool.GetExchange(),
				currentAmount.Token,
				currentAmount.Amount,
				p.Tokens[i+1].Address,
				err,
			)
		}
		swapTokenAmountOut, gas := calcAmountOutResult.TokenAmountOut, calcAmountOutResult.Gas
		if swapTokenAmountOut == nil {
			return poolpkg.TokenAmount{}, 0, errors.Wrapf(
				ErrInvalidSwap,
				"[Path.CalcAmountOut] CalcAmountOut returns nil | poolAddress: [%s], exchange: [%s], tokenIn: [%s], amountIn: [%s], tokenOut: [%s]",
				pool.GetAddress(),
				pool.GetExchange(),
				currentAmount.Token,
				currentAmount.Amount,
				p.Tokens[i+1].Address,
			)
		}

		currentAmount = *swapTokenAmountOut
		totalGas += gas
	}

	return currentAmount, totalGas, nil
}

// Equals returns true when two paths have same token and pool in respective order
func (p *Path) Equals(other *Path) bool {
	if len(p.PoolAddresses) != len(other.PoolAddresses) || len(p.Tokens) != len(other.Tokens) {
		return false
	}

	if p.Input.Token != other.Input.Token || p.Output.Token != other.Output.Token {
		return false
	}

	for idx := range p.PoolAddresses {
		if p.Tokens[idx] != other.Tokens[idx] || p.PoolAddresses[idx] != other.PoolAddresses[idx] {
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

	newInput := poolpkg.TokenAmount{
		Token:     p.Input.Token,
		Amount:    new(big.Int).Add(p.Input.Amount, other.Input.Amount),
		AmountUsd: p.Input.AmountUsd + other.Input.AmountUsd,
	}

	newOutput := poolpkg.TokenAmount{
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

	return p.cmpTokenLen(other)
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
