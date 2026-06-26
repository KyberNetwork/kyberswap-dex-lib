package caliberprop

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeCaliberProp

	methodGetAllPairIds   = "getAllPairIds"
	methodGetPoolBalances = "getPoolBalances"
	methodBatchQuote      = "batchQuote"
	methodQuote           = "quote"

	defaultGas = 250000
)

var (
	maxFetchPairCount  = big.NewInt(4)
	pairConfigBaseSlot = common.HexToHash("6")

	sampleBps = []int{10, 50, 250, 500, 1000, 2000, 3000, 5000, 7000, 9000, 9900}

	ErrInvalidToken          = errors.New("invalid token")
	ErrZeroAmount            = errors.New("zero amount in")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
	ErrAmountInTooLarge      = errors.New("amount in exceeds snapshot ladder")
	ErrNoQuote               = errors.New("no quote available for direction")
)
