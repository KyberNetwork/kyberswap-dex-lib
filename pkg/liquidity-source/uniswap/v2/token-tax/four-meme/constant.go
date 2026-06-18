package fourmeme

const (
	Protocol = "fourmeme"

	factory = "0xca143ce32fe78f1f7019d7d551a6402fc5350c73"

	methodPair    = "pair"
	methodBuyTax  = "feeRateBuy"
	methodSellTax = "feeRateSell"
)

var baseTokens = map[string]struct{}{
	"0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c": {},
}
