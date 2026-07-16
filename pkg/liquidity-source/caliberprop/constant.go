package caliberprop

import (
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

	defaultGas = 121632
)

var (
	maxFetchPairCount  = big.NewInt(4)
	pairConfigBaseSlot = common.HexToHash("6")
)
