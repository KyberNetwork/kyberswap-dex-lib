package valueobject

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
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
