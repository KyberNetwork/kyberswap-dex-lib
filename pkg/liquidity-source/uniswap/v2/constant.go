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
	pairMethodGetReserves = "getReserves"

	genericMethodFee       = "fee"
	genericTemplatePool    = "pool"
	genericTemplateFactory = "factory"

	// token transfer-tax methods (Virtual and four.meme agent tokens)
	tokenMethodIsLiquidityPool = "isLiquidityPool"
	tokenMethodTotalBuyTax     = "totalBuyTaxBasisPoints"
	tokenMethodTotalSellTax    = "totalSellTaxBasisPoints"
	tokenMethodPair            = "pair"
	tokenMethodFeeRateBuy      = "feeRateBuy"
	tokenMethodFeeRateSell     = "feeRateSell"
)

var (
	routerAddressByExchange = map[string]string{ // used both as router and approval address
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

	// Each transfer-tax protocol only lives on a specific factory (this v2 package is reused by many
	// forks) and its agent token pairs with a fixed base token. A pool is a candidate only when both
	// match; the other token is then the agent token to probe.
	virtualFactories = map[string]struct{}{
		"0x8909dc15e40173ff4699343b6eb8132c65e18ec6": {}, // Uniswap V2 (Base)
		"0x5c69bee701ef814a2b6a3edd4b1652cb9cc5aa6f": {}, // Uniswap V2 (Ethereum)
	}
	virtualBaseTokens = map[string]struct{}{
		"0x0b3e328455c4059eeb9e3f84b5543f74e24e7e1b": {}, // VIRTUAL (Base)
		"0x44ff8620b8ca30902395a7bd3f2407e1a091bf73": {}, // VIRTUAL (Ethereum)
	}
	fourMemeFactories = map[string]struct{}{
		"0xca143ce32fe78f1f7019d7d551a6402fc5350c73": {}, // PancakeSwap V2 (BSC)
	}
	fourMemeBaseTokens = map[string]struct{}{
		"0xbb4cdb9cbd36b01bd1cbaebf2de08d9173bc095c": {}, // WBNB
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
