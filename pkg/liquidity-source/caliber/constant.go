package caliber

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeCaliber

	methodGetPoolBalances = "getPoolBalances"
	methodBatchQuote      = "batchQuote"
	methodQuote           = "quote"
	methodGetPairId       = "getPairId"

	defaultGas = 250000

	bpsDenominator = 10000
)

var sampleBps = []int{10, 50, 250, 500, 1000, 2000, 3000, 5000, 7000, 9000, 9900}

var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrZeroAmount       = errors.New("zero amount in")
	ErrPoolUnavailable  = errors.New("pool not quoteable at snapshot time")
	ErrAmountInTooLarge = errors.New("amount in exceeds snapshot ladder")
	ErrNoQuote          = errors.New("no quote available for direction")
)
