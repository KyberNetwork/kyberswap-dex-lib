package fourmeme

const (
	Protocol = "fourmeme"

	methodPair = "pair"

	methodFeeRate = "feeRate"
	methodBuyTax  = "feeRateBuy"
	methodSellTax = "feeRateSell"
)

var factories = map[string]struct{}{
	"0xca143ce32fe78f1f7019d7d551a6402fc5350c73": {}, // BSC
}

var baseTokens = map[string]struct{}{
	"0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c": {}, // WBNB
	"0x55d398326f99059ff775485246999027b3197955": {}, // USDT
	"0x61a10e8556bed032ea176330e7f17d6a12a10000": {}, // UUSD
	"0x8d0d000ee44948fc98c9b98a4fa4921476f08b0d": {}, // USD1
}
