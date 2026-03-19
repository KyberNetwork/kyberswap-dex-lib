package uniswapv4

import (
	"math"
	"math/big"

	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	v3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/native/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
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

const TickBase = 1.0001

var Q96 = math.Pow(2, 96)

func EstimateReservesFromTicks(sqrtPriceX96 *big.Int, ticks []Tick) (amt0, amt1 *big.Int) {
	if len(ticks) == 0 {
		return bignumber.ZeroBI, bignumber.ZeroBI
	}
	var L, totalAmt0, totalAmt1 float64
	price, _ := sqrtPriceX96.Float64()
	price /= Q96

	upper := math.Pow(TickBase, float64(ticks[0].Index)/2)
	for i := 1; i < len(ticks); i++ {
		tickLower, tickUpper := ticks[i-1], ticks[i]
		liqNet, _ := tickLower.LiquidityNet.Float64()
		L += liqNet

		lower := upper
		upper = math.Pow(TickBase, float64(tickUpper.Index)/2)

		if price < lower {
			totalAmt0 += L * (upper - lower) / (lower * upper)
		} else if price >= upper {
			totalAmt1 += L * (upper - lower)
		} else {
			totalAmt0 += L * (upper - price) / (price * upper)
			totalAmt1 += L * (price - lower)
		}
	}

	var tmp big.Float
	amt0, _ = tmp.SetFloat64(totalAmt0).Int(new(big.Int))
	amt1, _ = tmp.SetFloat64(totalAmt1).Int(new(big.Int))
	return amt0, amt1
}

func EstimateReservesFromTicksU256(sqrtPriceX96 *uint256.Int, ticks []v3.TickU256) (amt0, amt1 *uint256.Int) {
	if len(ticks) == 0 {
		return big256.U0, big256.U0
	}
	var L, totalAmt0, totalAmt1 float64
	price := sqrtPriceX96.Float64() / Q96

	upper := math.Pow(TickBase, float64(ticks[0].Index)/2)
	for i := 1; i < len(ticks); i++ {
		tickLower, tickUpper := ticks[i-1], ticks[i]
		liqNet := ((*uint256.Int)(tickLower.LiquidityNet)).Float64()
		L += liqNet

		lower := upper
		upper = math.Pow(TickBase, float64(tickUpper.Index)/2)

		if price < lower {
			totalAmt0 += L * (upper - lower) / (lower * upper)
		} else if price >= upper {
			totalAmt1 += L * (upper - lower)
		} else {
			totalAmt0 += L * (upper - price) / (price * upper)
			totalAmt1 += L * (price - lower)
		}
	}

	var tmp big.Float
	amt0BI, _ := tmp.SetFloat64(totalAmt0).Int(new(big.Int))
	amt1BI, _ := tmp.SetFloat64(totalAmt1).Int(new(big.Int))
	return uint256.MustFromBig(amt0BI), uint256.MustFromBig(amt1BI)
}
