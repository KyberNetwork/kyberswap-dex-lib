package ethervista

const (
	DexType = "ether-vista"
	WETH    = "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2"
)

var (
	defaultGas = Gas{Swap: 60000}
)

const (
	factoryMethodAllPairs       = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodRouter         = "router"
)

const (
	pairMethodToken0       = "token0"
	pairMethodToken1       = "token1"
	pairMethodGetReserves  = "getReserves"
	pairMethodBuyTotalFee  = "buyTotalFee"
	pairMethodSellTotalFee = "sellTotalFee"
)

const (
	routerMethodUSDCToEth = "usdcToEth"
)
