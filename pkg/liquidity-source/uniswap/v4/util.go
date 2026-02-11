package uniswapv4

import (
	"errors"
	"math/big"

	"github.com/KyberNetwork/elastic-go-sdk/v2/utils"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func GetHookExchange(p *entity.Pool) valueobject.Exchange {
	var staticExtra StaticExtra
	var hookAddress common.Address
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		logger.Errorf("failed to unmarshal static extra data")
	} else {
		hookAddress = staticExtra.HooksAddress
	}

	hook, _ := GetHook(hookAddress, nil)
	return valueobject.Exchange(hook.GetExchange())
}

func CalculateReservesFromTicks(sqrtPriceX96 *big.Int, ticks []Tick) (*big.Int, *big.Int, error) {
	L := big.NewInt(0)
	totalAmount0, totalAmount1 := big.NewInt(0), big.NewInt(0)

	for i, tickLower := range ticks {
		L.Add(L, tickLower.LiquidityNet)

		if L.Sign() == 0 {
			continue
		}

		if i == len(ticks)-1 {
			return nil, nil, errors.New("sum liquidity net is not zero")
		}

		tickUpper := ticks[i+1]

		sqrtLower, err := utils.GetSqrtRatioAtTick(tickLower.Index)
		if err != nil {
			return nil, nil, err
		}
		sqrtUpper, err := utils.GetSqrtRatioAtTick(tickUpper.Index)
		if err != nil {
			return nil, nil, err
		}

		var numer, denom, amount0, amount1, tmp big.Int
		if sqrtPriceX96.Cmp(sqrtLower) < 0 {
			numer.Mul(L, Q96).Mul(&numer, tmp.Sub(sqrtUpper, sqrtLower))
			denom.Mul(sqrtLower, sqrtUpper)

			amount0.Div(&numer, &denom)
		} else if sqrtPriceX96.Cmp(sqrtUpper) >= 0 {
			numer.Mul(L, tmp.Sub(sqrtUpper, sqrtLower))

			amount1.Div(&numer, Q96)
		} else {
			numer.
				Mul(L, Q96).
				Mul(&numer, tmp.Sub(sqrtUpper, sqrtPriceX96))
			denom.Mul(sqrtPriceX96, sqrtUpper)
			amount0.Div(&numer, &denom)

			numer.Mul(L, tmp.Sub(sqrtPriceX96, sqrtLower))
			amount1.Div(&numer, Q96)
		}

		totalAmount0.Add(totalAmount0, &amount0)
		totalAmount1.Add(totalAmount1, &amount1)
	}

	return totalAmount0, totalAmount1, nil
}
