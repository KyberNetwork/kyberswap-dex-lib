package business

import (
	"context"
	"fmt"
	"math/big"

	"github.com/KyberNetwork/blockchain-toolkit/number"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	composablestable "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v2/composable-stable"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/limitorder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/maverickv1"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/synthetix"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/pkg/logger"
)

func CalculatePoolTVL(
	ctx context.Context,
	p *entity.Pool,
	nativePriceByToken map[string]*routerEntity.OnchainPrice,
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
				midPrice, price, err := getMidPrice(nativePriceByToken, poolTokens[i].Address)
				if err != nil {
					logger.Debugf(ctx, "cannot get mid price for token %v %v", poolTokens[i], price)
					return 0, err
				}

				reserveBF, err := getReserve(ctx, p, i, price.Decimals)
				if err != nil {
					return 0, err
				}

				// we're using `NativePriceRaw` so no need to divide to token's 10^decimals
				rawNativeWei := new(big.Float).Mul(reserveBF, midPrice)
				nativeValue, _ := new(big.Float).Quo(rawNativeWei, constant.BoneFloat).Float64()

				logger.Debugf(ctx, "reserve %v price %v value %v", reserveBF, midPrice, nativeValue)
				reserveNative += nativeValue
			}

			return reserveNative, nil
		}
	}
}

func getReserve(ctx context.Context, p *entity.Pool, i int, decimals uint8) (*big.Float, error) {
	switch p.Type {
	case maverickv1.DexTypeMaverickV1:
		// maverick's reserves need to be scaled up/down first
		reserveRaw, err := maverickv1.ScaleToAmount(number.NewUint256(p.Reserves[i]), decimals)
		if err != nil {
			logger.Debugf(ctx, "invalid pool reserve %v %v", p.Address, p.Reserves[i])
			return nil, ErrorInvalidReserve
		}

		reserveBF, ok := new(big.Float).SetString(reserveRaw.String())
		if !ok {
			return nil, fmt.Errorf("fail to convert pool reserve to big float: %v", p.Reserves[i])
		}

		return reserveBF, nil

	case composablestable.DexType:
		// need to ignore the pool token itself
		if p.Tokens[i].Address == p.Address {
			return big.NewFloat(0), nil
		}
		if reserveBF, ok := new(big.Float).SetString(p.Reserves[i]); !ok {
			logger.Errorf(ctx, "invalid pool reserve %v %v", p.Address, p.Reserves[i])
			return nil, ErrorInvalidReserve
		} else {
			return reserveBF, nil
		}

	default:
		if reserveBF, ok := new(big.Float).SetString(p.Reserves[i]); !ok {
			logger.Errorf(ctx, "invalid pool reserve %v %v", p.Address, p.Reserves[i])
			return nil, ErrorInvalidReserve
		} else {
			return reserveBF, nil
		}
	}
}

// we'll use mid price (or buy price if missing sell price) to calculate TVL
func getMidPrice(nativePriceByToken map[string]*routerEntity.OnchainPrice, token string) (*big.Float, *routerEntity.OnchainPrice, error) {
	tokenNativePrice, ok := nativePriceByToken[token]
	if !ok {
		return nil, nil, ErrorNoPrice
	}

	midPrice := tokenNativePrice.NativePriceRaw.Buy
	if tokenNativePrice.NativePriceRaw.Buy != nil && tokenNativePrice.NativePriceRaw.Sell != nil {
		midPrice = new(big.Float).Quo(
			new(big.Float).Add(tokenNativePrice.NativePriceRaw.Buy, tokenNativePrice.NativePriceRaw.Sell),
			big.NewFloat(2))
	}
	if midPrice == nil {
		return nil, tokenNativePrice, ErrorNoPrice
	}
	return midPrice, tokenNativePrice, nil
}
