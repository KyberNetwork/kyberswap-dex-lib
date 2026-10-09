package prismprop

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangePrismProp

	methodGetSupportedPairs      = "getSupportedPairs"
	methodGetOrderBookWithParams = "getOrderBookWithParams"
)

// executorAddress is the ExecutorHelper that calls router.swap() in a live
// route. The engine prices by msg.sender, so the book is quoted for it.
var executorAddress = common.HexToAddress("0x8f10b468b06c6fd214b65f87778827f7d113f996")
