package uniswapv4

import (
	"math"
	"math/big"
	"strings"

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
	return valueobject.Exchange(ExchangeOf(hook, "", p.Exchange))
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

		if price <= lower {
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
		liqNet := big256.SFloat64(tickLower.LiquidityNet)
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

	var tmpBF big.Float
	var tmpBI big.Int
	var overflow bool
	amt0BI, _ := tmpBF.SetFloat64(totalAmt0).Int(&tmpBI)
	if amt0, overflow = uint256.FromBig(amt0BI); overflow {
		amt0 = new(uint256.Int)
	}
	amt1BI, _ := tmpBF.SetFloat64(totalAmt1).Int(&tmpBI)
	if amt1, overflow = uint256.FromBig(amt1BI); overflow {
		amt1 = new(uint256.Int)
	}
	return amt0, amt1
}

// ForkExchange swaps the "uniswap-v4" prefix of a hook exchange for dexID ("uniswap-v4-fee" ->
// "elyspump-v4-fee"). Empty or official dexID, and hook exchanges without the prefix (kem), stay as is.
func ForkExchange(hookExchange, dexID string) string {
	if dexID == "" || dexID == DexType || !strings.HasPrefix(hookExchange, DexType) {
		return hookExchange
	}
	return dexID + hookExchange[len(DexType):]
}

// ExchangeOf is the exchange of a pool with this hook. The dexID is cfgDexID, else recovered from
// the stored exchange by stripping the hook suffix; a stored "uniswap-v4*" value means the official dex.
func ExchangeOf(hook Hook, cfgDexID, stored string) string {
	h := hook.GetExchange()
	if cfgDexID == "" && strings.HasPrefix(h, DexType) {
		cfgDexID, _ = strings.CutSuffix(stored, h[len(DexType):])
	}
	return ForkExchange(h, cfgDexID)
}

// PoolManagerOf is the pool's own PoolManager from StaticExtra, else the official one for the chain.
func PoolManagerOf(p *entity.Pool, chainID valueobject.ChainID) common.Address {
	var se StaticExtra
	if p != nil && json.Unmarshal([]byte(p.StaticExtra), &se) == nil && se.PoolManager != nil {
		return *se.PoolManager
	}
	return PoolManager[chainID]
}
