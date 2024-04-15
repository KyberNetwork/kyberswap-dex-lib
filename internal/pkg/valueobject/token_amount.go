package valueobject

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

// this is used inside router-service and should use token unit and native unit only (without usd)
type TokenAmount struct {
	Token          string
	Amount         *big.Int // in token unit
	AmountAfterGas *big.Int // in native unit

	AmountUsd float64 // will be deprecated later after we fully switch to onchain-price-service
}

func (a *TokenAmount) String() string {
	return fmt.Sprintf("TokenAmount(%v, %v, %v, %v)", a.Token, a.Amount, a.AmountAfterGas, a.AmountUsd)
}

// CompareTo in old dex-lib TokenAmount
func (a *TokenAmount) CompareRaw(b *TokenAmount) int {
	if b == nil || b.Token != a.Token {
		return -1
	}
	return a.Amount.Cmp(b.Amount)
}

// return 1 if a greater than b, 0 if a == b, -1 otherwise
func (a *TokenAmount) Compare(b *TokenAmount, gasFeeInclude bool) int {
	if a == nil && b == nil {
		return 0
	}
	if a == nil {
		return -1
	}
	if b == nil {
		return 1
	}

	// If we consider gas fee, prioritize node with more AmountUsd
	// If amountUsd is the same, compare amountOut regardless of gasFeeInclude
	if gasFeeInclude {
		// if we're using amount in native unit
		if a.AmountAfterGas != nil && b.AmountAfterGas != nil {
			cmp := a.AmountAfterGas.Cmp(b.AmountAfterGas)
			if cmp != 0 {
				return cmp
			}
		}
		// otherwise compare amount in usd
		if !utils.Float64AlmostEqual(a.AmountUsd, b.AmountUsd) {
			if a.AmountUsd > b.AmountUsd {
				return 1
			} else {
				return -1
			}
		}
	}
	// Otherwise, prioritize node with more token Amount
	return a.Amount.Cmp(b.Amount)
}

func (a *TokenAmount) ToDexLibAmount() *pool.TokenAmount {
	return &pool.TokenAmount{
		Token:     a.Token,
		Amount:    new(big.Int).Set(a.Amount),
		AmountUsd: a.AmountUsd,
	}
}

func FromDexLibAmount(a *pool.TokenAmount) *TokenAmount {
	return &TokenAmount{
		Token:     a.Token,
		Amount:    new(big.Int).Set(a.Amount),
		AmountUsd: a.AmountUsd,
	}
}
