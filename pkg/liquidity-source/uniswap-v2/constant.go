package uniswapv2

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "uniswap-v2"

	defaultGas = 76562

	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"

	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodReserve0    = "reserve0"
	pairMethodReserve1    = "reserve1"
	pairMethodGetReserves = "getReserves"

	genericMethodFee       = "fee"
	genericTemplatePool    = "pool"
	genericTemplateFactory = "factory"
)

var (
	approvalAddressByExchange = map[string]string{ // used both as approval and router address
		valueobject.ExchangeBabyDogeSwap: "0xC9a0F685F39d05D835c369036251ee3aEaaF3c47",
		valueobject.ExchangeBakerySwap:   "0xCDe540d7eAFE93aC5fE6233Bee57E1270D3E330F",
		valueobject.ExchangeMeshSwap:     "0x10f4A785F458Bc144e3706575924889954946639",
	}
	extraGasByExchange = map[string]int64{
		valueobject.ExchangeBabyDogeSwap: 259957 - defaultGas,
		valueobject.ExchangeBakerySwap:   111012 - defaultGas,
		valueobject.ExchangeMeshSwap:     321758 - defaultGas,
	}
	noFOTByExchange = map[string]bool{ // these exchanges don't support FOT
		valueobject.ExchangeMeshSwap: true,
	}

	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInvalidAmountOut         = errors.New("invalid amount out")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrInsufficientLiquidity    = errors.New("INSUFFICIENT_LIQUIDITY")
	ErrInvalidK                 = errors.New("K")
)
