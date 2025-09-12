package business

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	algebrav1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/algebra/v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ambient"
	maverickv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maverick/v1"
	pancakev3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/v3"
	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/iziswap"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv20"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/liquiditybookv21"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ramsesv2"
	solidlyv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/solidly-v3"
	"github.com/goccy/go-json"
	"github.com/izumiFinance/iZiSwap-SDK-go/library/calc"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func CalculatePoolAmplifiedTVL(
	ctx context.Context,
	p *entity.Pool,
	nativePriceByToken map[string]*routerEntity.OnchainPrice,
) (float64, bool, error) {
	poolTokens := p.Tokens
	switch p.Type {
	case uniswapv3.DexTypeUniswapV3,
		solidlyv3.DexTypeSolidlyV3,
		ramsesv2.DexTypeRamsesV2,
		pancakev3.DexTypePancakeV3,
		maverickv1.DexTypeMaverickV1,
		algebrav1.DexTypeAlgebraV1,
		iziswap.DexTypeiZiSwap,
		liquiditybookv20.DexTypeLiquidityBookV20, liquiditybookv21.DexTypeLiquidityBookV21:
		liquidityF, sqrtPriceF, err := getLiquidityAndSqrtPrice(p)
		if err != nil {
			log.Ctx(ctx).Debug().Err(err).Msgf("failed to get liquidity and sqrt price for pool %s", p.Address)
			return 0, false, err
		}

		v, err := calculateAmplifiedTVL(ctx, poolTokens[0].Address, poolTokens[1].Address, liquidityF, sqrtPriceF,
			nativePriceByToken)
		return v, false, err

	case ambient.DexTypeAmbient:
		var staticExtra ambient.StaticExtra
		if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
			return 0, false, fmt.Errorf("could not unmarshal ambient.StaticExtra: %w", err)
		}

		var extra ambient.Extra
		if err := json.Unmarshal([]byte(p.Extra), &extra); err != nil {
			return 0, false, fmt.Errorf("could not unmarshal ambient.Extra: %w", err)
		}

		var amplifiedTvl float64

		for pair, pairInfo := range extra.TokenPairs {
			token0, token1 := pair.Base, pair.Quote
			if token0 == ambient.NativeTokenPlaceholderAddress {
				token0 = staticExtra.NativeTokenAddress
			}
			if token1 == ambient.NativeTokenPlaceholderAddress {
				token1 = staticExtra.NativeTokenAddress
			}

			liquidityF, _ := strconv.ParseFloat(pairInfo.Liquidity, 64)
			sqrtPriceF, _ := strconv.ParseFloat(pairInfo.SqrtPriceX64, 64)

			v, err := calculateAmplifiedTVL(ctx,
				strings.ToLower(token0.String()),
				strings.ToLower(token1.String()),
				liquidityF,
				sqrtPriceF,
				nativePriceByToken,
			)
			if err != nil {
				log.Ctx(ctx).Warn().Msgf("could not CalculateAmplifiedTVL for Ambient pair %s", pair)
				continue
			}
			amplifiedTvl += v
		}

		return amplifiedTvl, false, nil

	default:
		if p.HasReserves() {
			// return true to use pool's TVL
			return 0, true, nil
		}
		return 0, false, nil
	}
}

func calculateAmplifiedTVL(
	ctx context.Context,
	token0, token1 string,
	liquidityF, sqrtPriceF float64,
	nativePriceByToken map[string]*routerEntity.OnchainPrice,
) (float64, error) {
	if liquidityF == 0.0 || sqrtPriceF == 0.0 {
		return 0.0, nil
	}

	midPrice0 := nativePriceByToken[token0].GetMidPriceNativeRaw()
	if midPrice0 == 0.0 {
		log.Ctx(ctx).Debug().Msgf("cannot get mid price for token0 %v", token0)
		return 0, fmt.Errorf("token has no price %s", token0)
	}

	midPrice1 := nativePriceByToken[token1].GetMidPriceNativeRaw()
	if midPrice1 == 0.0 {
		log.Ctx(ctx).Debug().Msgf("cannot get mid price for token1 %v", token1)
		return 0, fmt.Errorf("token has no price %s", token1)
	}

	// Formula: amplifiedTvl = priceOfXinUSD*Liquidity/SqrtPrice + Liquidity*SqrtPrice*priceOfYinUSD
	// Doc: https://www.notion.so/kybernetwork/Aggregator-Uniswap-v3-Integration-technical-design-f746167703c448dcaa40f523301e11b4?pvs=4#bd82e866196141dc97566440483afa47

	// first get the 2 virtual reserves
	virtualRev0 := liquidityF / sqrtPriceF
	virtualRev1 := liquidityF * sqrtPriceF

	// we're using `NativePriceRaw` so no need to divide to token's 10^decimals
	virtualRev0 = (virtualRev0 * midPrice0) / constant.BoneFloat64
	virtualRev1 = (virtualRev1 * midPrice1) / constant.BoneFloat64

	return virtualRev0 + virtualRev1, nil
}

// this should return raw sqrtPrice instead of encoded price (x96, d18...)
func getLiquidityAndSqrtPrice(p *entity.Pool) (float64, float64, error) {
	var liquidity *big.Int
	var sqrtPrice *big.Float

	switch p.Type {
	case uniswapv3.DexTypeUniswapV3:
		extra := uniswapv3.Extra{}
		var err = json.Unmarshal([]byte(p.Extra), &extra)
		if err != nil {
			return 0.0, 0.0, err
		}

		liquidity, sqrtPrice = extra.Liquidity, fromSqrtPriceX96(extra.SqrtPriceX96)
	case solidlyv3.DexTypeSolidlyV3:
		extra := solidlyv3.Extra{}
		var err = json.Unmarshal([]byte(p.Extra), &extra)
		if err != nil {
			return 0.0, 0.0, err
		}

		liquidity, sqrtPrice = extra.Liquidity, fromSqrtPriceX96(extra.SqrtPriceX96)
	case ramsesv2.DexTypeRamsesV2:
		extra := ramsesv2.Extra{}
		var err = json.Unmarshal([]byte(p.Extra), &extra)
		if err != nil {
			return 0.0, 0.0, err
		}

		liquidity, sqrtPrice = extra.Liquidity, fromSqrtPriceX96(extra.SqrtPriceX96)
	case pancakev3.DexTypePancakeV3:
		extra := pancakev3.Extra{}
		var err = json.Unmarshal([]byte(p.Extra), &extra)
		if err != nil {
			return 0.0, 0.0, err
		}

		liquidity, sqrtPrice = extra.Liquidity, fromSqrtPriceX96(extra.SqrtPriceX96)
	case maverickv1.DexTypeMaverickV1:
		extra := maverickv1.Extra{}
		var err = json.Unmarshal([]byte(p.Extra), &extra)
		if err != nil {
			return 0.0, 0.0, err
		}

		// maverick actually used D18 representation (sqrtPrice * 1e18)
		liquidity, sqrtPrice = extra.Liquidity.ToBig(), fromSqrtPriceD18(extra.SqrtPriceX96.ToBig())
	case algebrav1.DexTypeAlgebraV1:
		extra := algebrav1.Extra{}
		var err = json.Unmarshal([]byte(p.Extra), &extra)
		if err != nil {
			return 0.0, 0.0, err
		}

		liquidity, sqrtPrice = extra.Liquidity, fromSqrtPriceX96(extra.GlobalState.Price)
	case liquiditybookv21.DexTypeLiquidityBookV21:
		extra := liquiditybookv21.Extra{}
		var err = json.Unmarshal([]byte(p.Extra), &extra)
		if err != nil {
			return 0.0, 0.0, err
		}

		liquidity, sqrtPrice = extra.Liquidity, fromSqrtPriceX128(extra.PriceX128)

	case iziswap.DexTypeiZiSwap:
		extra := iziswap.Extra{}
		var err = json.Unmarshal([]byte(p.Extra), &extra)
		if err != nil {
			return 0.0, 0.0, err
		}

		sqrtPriceX96, err := calc.GetSqrtPrice(extra.CurrentPoint)
		if err != nil {
			return 0.0, 0.0, err
		}

		liquidity, sqrtPrice = extra.Liquidity, fromSqrtPriceX96(sqrtPriceX96)

	}

	if liquidity == nil {
		return 0.0, 0.0, ErrNilLiquidity
	}

	if sqrtPrice == nil {
		return 0.0, 0.0, ErrNilSqrtPrice
	}

	sqrtPriceF, _ := sqrtPrice.Float64()
	liquidityF, _ := liquidity.Float64()

	return liquidityF, sqrtPriceF, nil
}

var (
	X128, _ = new(big.Int).SetString("100000000000000000000000000000000", 16)
	X96, _  = new(big.Int).SetString("1000000000000000000000000", 16)
	X64, _  = new(big.Int).SetString("10000000000000000", 16)
	D18, _  = new(big.Int).SetString("1000000000000000000", 10)
	X128BF  = new(big.Float).SetInt(X128)
	X96BF   = new(big.Float).SetInt(X96)
	X64BF   = new(big.Float).SetInt(X64)
	D18BF   = new(big.Float).SetInt(D18)
)

func fromSqrtPriceX96(sqrtPrice *big.Int) *big.Float {
	if sqrtPrice == nil {
		return nil
	}
	return new(big.Float).Quo(new(big.Float).SetInt(sqrtPrice), X96BF)
}

func fromSqrtPriceD18(sqrtPrice *big.Int) *big.Float {
	if sqrtPrice == nil {
		return nil
	}
	return new(big.Float).Quo(new(big.Float).SetInt(sqrtPrice), D18BF)
}

func fromSqrtPriceX128(sqrtPrice *big.Int) *big.Float {
	if sqrtPrice == nil {
		return nil
	}
	return new(big.Float).Quo(new(big.Float).SetInt(sqrtPrice), X128BF)
}
