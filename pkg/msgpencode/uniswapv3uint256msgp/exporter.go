package uniswapv3uint256msgp

import (
	"unsafe"

	"github.com/KyberNetwork/uniswapv3-sdk-uint256/constants"
	uniswapv3entities "github.com/KyberNetwork/uniswapv3-sdk-uint256/entities"
	"github.com/KyberNetwork/uniswapv3-sdk-uint256/utils"
	"github.com/daoleno/uniswap-sdk-core/entities"
)

func init() {
	if unsafe.Sizeof(poolExporter{}) != unsafe.Sizeof(uniswapv3entities.Pool{}) {
		panic("Sizeof(poolExporter) must equal to Sizeof(pancakev3entities.Pool)")
	}

	if unsafe.Sizeof(nativeExporter{}) != unsafe.Sizeof(entities.Native{}) {
		panic("Sizeof(nativeExporter) must equal to Sizeof(entities.Native)")
	}

	if unsafe.Sizeof(baseCurrencyExporter{}) != unsafe.Sizeof(entities.BaseCurrency{}) {
		panic("Sizeof(baseCurrencyExporter) must equal to Sizeof(entities.BaseCurrency)")
	}

	if unsafe.Sizeof(tickListDataProviderExporter{}) != unsafe.Sizeof(uniswapv3entities.TickListDataProvider{}) {
		panic("Sizeof(tickListDataProviderExporter) must equal to Sizeof(pancakev3entities.TickListDataProvider)")
	}
}

// nativeExporter has the same structure as entities.Native
type nativeExporter struct {
	*entities.BaseCurrency
	wrapped *entities.Token
}

func exportNative(n *entities.Native) *nativeExporter {
	return (*nativeExporter)(unsafe.Pointer(n))
}

// poolExporter has the same structure as uniswapv3entities.Pool
type poolExporter struct {
	Token0           *entities.Token
	Token1           *entities.Token
	Fee              constants.FeeAmount
	SqrtRatioX96     *utils.Uint160
	Liquidity        *utils.Uint128
	TickCurrent      int
	TickDataProvider uniswapv3entities.TickDataProvider

	token0Price *entities.Price
	token1Price *entities.Price
}

func exportPool(pool *uniswapv3entities.Pool) *poolExporter {
	return (*poolExporter)(unsafe.Pointer(pool))
}

// baseCurrencyExporter has the same structure as entities.BaseCurrency
type baseCurrencyExporter struct {
	currency entities.Currency
	isNative bool   // Returns whether the currency is native to the chain and must be wrapped (e.g. Ether)
	isToken  bool   // Returns whether the currency is a token that is usable in Uniswap without wrapping
	chainId  uint   // The chain ID on which this currency resides
	decimals uint   // The decimals used in representing currency amounts
	symbol   string // The symbol of the currency, i.e. a short textual non-unique identifier
	name     string // The name of the currency, i.e. a descriptive textual non-unique identifier
}

// tickListDataProviderExporter has the same structure as entities.TickListDataProvider
func fromBaseCurrencyExporter(b *baseCurrencyExporter) *entities.BaseCurrency {
	return (*entities.BaseCurrency)(unsafe.Pointer(b))
}

func exportBaseCurrency(sdk *entities.BaseCurrency) *baseCurrencyExporter {
	return (*baseCurrencyExporter)(unsafe.Pointer(sdk))
}

type tickListDataProviderExporter struct {
	ticks []uniswapv3entities.Tick
}

func exportTickListDataProvider(t *uniswapv3entities.TickListDataProvider) *tickListDataProviderExporter {
	return (*tickListDataProviderExporter)(unsafe.Pointer(t))
}
