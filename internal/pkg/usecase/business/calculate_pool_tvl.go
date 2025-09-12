package business

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	composablestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v2/composable-stable"
	maverickv1 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maverick/v1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/synthetix"
	"github.com/rs/zerolog/log"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func CalculatePoolTVL(
	ctx context.Context,
	p *entity.Pool,
	nativePriceByToken map[string]*routerEntity.OnchainPrice,
	partialTvl bool,
) (float64, error) {
	poolTokens := p.Tokens

	switch p.Type {
	case limitorder.DexTypeLimitOrder:
		// Currently, the total of TVL/reserveUsd in the limit order pool will be very small compared with other pools. So it will be filtered in choosing pools process
		// We will use big hardcode number to push it into eligible pools for findRoute algorithm.
		// (this is in USD, not native, but it's still ok because we only need this to push LO pools up the ranking)
		return limitorder.LimitOrderPoolReserveUSD, nil

	case synthetix.DexTypeSynthetix:
		{
			// we no longer support Synthetix, will add code here if we ever do
			return 0, nil
		}

	default:
		{
			var reserveNative = float64(0)
			for i := range poolTokens {
				token := poolTokens[i].Address
				tokenNativePrice, ok := nativePriceByToken[token]

				if !ok {
					if partialTvl {
						continue
					}
					return 0, fmt.Errorf("token has no price %s", token)
				}

				reserveF, err := getReserve(ctx, p, i, tokenNativePrice.Decimals)
				if err != nil {
					return 0, err
				}
				if reserveF == 0.0 {
					continue
				}

				midPriceF := nativePriceByToken[poolTokens[i].Address].GetMidPriceNativeRaw()
				if midPriceF == 0.0 {
					// we need partially calculate tvl if some tokens in a pool have no price in case liquidity score ranking
					if partialTvl {
						log.Ctx(ctx).Error().Msgf("cannot get mid price for token %v pool %v type %v",
							poolTokens[i].Address, p.Address, p.Type)
						continue
					}
					return 0, fmt.Errorf("token has no price %s", poolTokens[i].Address)
				}

				// we're using `NativePriceRaw` so no need to divide to token's 10^decimals
				nativeValue := (reserveF * midPriceF) / constant.BoneFloat64

				log.Ctx(ctx).Debug().Msgf("reserve %v price %v value %v", reserveF, midPriceF, nativeValue)
				reserveNative += nativeValue
			}

			return reserveNative, nil
		}
	}
}

func CalculatePoolTVLForTokenPair(
	ctx context.Context,
	p *entity.Pool,
	nativePriceByToken map[string]*routerEntity.OnchainPrice,
	indexes []int,
) (float64, error) {
	poolTokens := p.Tokens

	switch p.Type {
	case limitorder.DexTypeLimitOrder:
		// Currently, the total of TVL/reserveUsd in the limit order pool will be very small compared with other pools. So it will be filtered in choosing pools process
		// We will use big hardcode number to push it into eligible pools for findRoute algorithm.
		// (this is in USD, not native, but it's still ok because we only need this to push LO pools up the ranking)
		return limitorder.LimitOrderPoolReserveUSD, nil

	case synthetix.DexTypeSynthetix:
		{
			// we no longer support Synthetix, will add code here if we ever do
			return 0, nil
		}

	default:
		{
			var reserveNative = float64(0)
			for _, i := range indexes {
				if i < 0 || i >= len(poolTokens) {
					return 0.0, errors.New("index is invalid")
				}

				price := nativePriceByToken[poolTokens[i].Address]
				midPriceF := price.GetMidPriceNativeRaw()
				if midPriceF == 0.0 {
					return 0.0, fmt.Errorf("token has no price %s", poolTokens[i].Address)
				}

				reserveF, err := getReserve(ctx, p, i, price.Decimals)
				if err != nil {
					return 0.0, err
				}

				// we're using `NativePriceRaw` so no need to divide to token's 10^decimals
				nativeValue := (reserveF * midPriceF) / constant.BoneFloat64

				log.Ctx(ctx).Debug().Msgf("reserve %v price %v value %v", reserveF, midPriceF, nativeValue)
				reserveNative += nativeValue
			}

			return reserveNative, nil
		}
	}
}

func getReserve(ctx context.Context, p *entity.Pool, i int, decimals uint8) (float64, error) {
	if i >= len(p.Reserves) {
		return 0.0, ErrorInvalidReserve
	}

	switch p.Type {
	case maverickv1.DexTypeMaverickV1:
		// maverick's reserves need to be scaled up/down first
		reserveRaw, err := maverickv1.ScaleToAmount(number.NewUint256(p.Reserves[i]), decimals)
		if err != nil {
			log.Ctx(ctx).Debug().Msgf("invalid pool reserve %v %v", p.Address, p.Reserves[i])
			return 0.0, ErrorInvalidReserve
		}

		reserveF, err := strconv.ParseFloat(reserveRaw.String(), 64)
		if err != nil {
			return 0.0, fmt.Errorf("fail to convert pool reserve to float64: %v", p.Reserves[i])
		}

		return reserveF, nil

	case composablestable.DexType:
		// need to ignore the pool token itself
		if p.Tokens[i].Address == p.Address {
			return 0.0, nil
		}
		if reserveF, err := strconv.ParseFloat(p.Reserves[i], 64); err != nil {
			log.Ctx(ctx).Error().Msgf("invalid pool reserve %v %v", p.Address, p.Reserves[i])
			return 0.0, ErrorInvalidReserve
		} else {
			return reserveF, nil
		}

	default:
		if reserveF, err := strconv.ParseFloat(p.Reserves[i], 64); err != nil {
			log.Ctx(ctx).Error().Msgf("invalid pool reserve %v %v", p.Address, p.Reserves[i])
			return 0.0, ErrorInvalidReserve
		} else {
			return reserveF, nil
		}
	}
}
