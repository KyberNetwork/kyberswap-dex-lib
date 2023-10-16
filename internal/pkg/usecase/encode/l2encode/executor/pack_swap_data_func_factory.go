package executor

import (
	"github.com/pkg/errors"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/executor/swapdata"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	packSwapDataFuncRegistry           = map[valueobject.Exchange]PackSwapDataFunc{}
	ErrPackSwapDataFuncIsNotRegistered = errors.New("pack swap data function is not registered")
)

// PackSwapDataFunc is a function to pack swap data
type PackSwapDataFunc func(chainID valueobject.ChainID, swap types.L2EncodingSwap) ([]byte, error)

func RegisterPackSwapDataFunc(exchange valueobject.Exchange, fn PackSwapDataFunc) {
	packSwapDataFuncRegistry[exchange] = fn
}

func GetPackSwapDataFunc(exchange valueobject.Exchange) (PackSwapDataFunc, error) {
	fn, ok := packSwapDataFuncRegistry[exchange]
	if !ok {
		return nil, errors.Wrapf(ErrPackSwapDataFuncIsNotRegistered, "exchange: [%s]", exchange)
	}

	return fn, nil
}

func init() {
	// Uniswap
	RegisterPackSwapDataFunc(valueobject.ExchangeCamelot, swapdata.PackCamelot) // Custom PackUniswap
	RegisterPackSwapDataFunc(valueobject.ExchangeChronos, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeFraxSwap, swapdata.PackFraxSwap) // Custom PackUniswap
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberSwap, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberSwapStatic, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangePancake, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeRamses, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSushiSwap, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSwapr, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeUniSwap, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeVelodrome, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeVelodromeV2, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeZipSwap, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSpartaDex, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeArbiDex, swapdata.PackBiswap) // Custom PackUniswap
	RegisterPackSwapDataFunc(valueobject.ExchangeBaseSwap, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeAlienBase, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSwapBased, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeBvm, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeBaso, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSynthSwap, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeRocketSwapV2, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeAerodrome, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeDackieV2, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeMoonBase, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeScale, swapdata.PackUniswap)
	RegisterPackSwapDataFunc(valueobject.ExchangeBalDex, swapdata.PackUniswap)

	// StableSwap
	RegisterPackSwapDataFunc(valueobject.ExchangeSaddle, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeSynapse, swapdata.PackStableSwap)
	RegisterPackSwapDataFunc(valueobject.ExchangeAlienBaseStableSwap, swapdata.PackStableSwap)

	// CurveSwap
	RegisterPackSwapDataFunc(valueobject.ExchangeCurve, swapdata.PackCurveSwap)

	// UniSwapV3ProMM
	RegisterPackSwapDataFunc(valueobject.ExchangeChronosV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberswapElastic, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangePancakeV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeRamsesV2, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeSushiSwapV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeUniSwapV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeArbiDexV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeMMFV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeHorizonDex, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeDackieV3, swapdata.PackUniswapV3KSElastic)
	RegisterPackSwapDataFunc(valueobject.ExchangeBaseSwapV3, swapdata.PackUniswapV3KSElastic)

	// BalancerV2
	RegisterPackSwapDataFunc(valueobject.ExchangeBalancer, swapdata.PackBalancerV2)
	RegisterPackSwapDataFunc(valueobject.ExchangeBalancerComposableStable, swapdata.PackBalancerV2)
	RegisterPackSwapDataFunc(valueobject.ExchangeBeethovenX, swapdata.PackBalancerV2)

	// DODO
	RegisterPackSwapDataFunc(valueobject.ExchangeDodo, swapdata.PackDODO)

	// GMX
	RegisterPackSwapDataFunc(valueobject.ExchangeGMX, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeBMX, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeSynthSwapPerp, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeSwapBasedPerp, swapdata.PackGMX)
	RegisterPackSwapDataFunc(valueobject.ExchangeBMXGLP, swapdata.PackGmxGlp)

	// KyberLimitOrder
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberSwapLimitOrder, swapdata.PackKyberLimitOrder)
	RegisterPackSwapDataFunc(valueobject.ExchangeKyberSwapLimitOrderDS, swapdata.PackKyberLimitOrderDS)

	// MaverickV1
	RegisterPackSwapDataFunc(valueobject.ExchangeMaverickV1, swapdata.PackMaverickV1)

	// AlgebraV1
	RegisterPackSwapDataFunc(valueobject.ExchangeCamelotV3, swapdata.PackAlgebraV1)
	RegisterPackSwapDataFunc(valueobject.ExchangeZyberSwapV3, swapdata.PackAlgebraV1)
	RegisterPackSwapDataFunc(valueobject.ExchangeSynthSwapV3, swapdata.PackAlgebraV1)
	RegisterPackSwapDataFunc(valueobject.ExchangeSwapBasedV3, swapdata.PackAlgebraV1)

	// IZiSwap
	RegisterPackSwapDataFunc(valueobject.ExchangeIZiSwap, swapdata.PackIZiSwap)

	// Wombat
	RegisterPackSwapDataFunc(valueobject.ExchangeWombat, swapdata.PackWombat)
	RegisterPackSwapDataFunc(valueobject.ExchangeWooFiV2, swapdata.PackWombat)
}
