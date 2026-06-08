package whlp

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

var (
	ErrAccountantPaused = errors.New("accountant is paused")
	ErrInvalidToken     = errors.New("invalid token for swap")
	ErrZeroAmount       = errors.New("zero amount")
)

func quoteToShare(amountIn, rate, oneShare *big.Int) (*big.Int, error) {
	if amountIn.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	if rate.Sign() <= 0 {
		return nil, ErrInvalidToken
	}
	var shares big.Int
	bignumber.MulDivDown(&shares, amountIn, oneShare, rate)
	return &shares, nil
}

func shareToQuote(amountIn, rate, oneShare *big.Int) (*big.Int, error) {
	if amountIn.Sign() <= 0 {
		return nil, ErrZeroAmount
	}
	if rate.Sign() <= 0 {
		return nil, ErrInvalidToken
	}
	var quote big.Int
	bignumber.MulDivDown(&quote, amountIn, rate, oneShare)
	return &quote, nil
}
