package stable

import (
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

func TestCloneState(t *testing.T) {
	t.Parallel()
	p, err := NewPoolSimulator(entity.Pool{
		Address:  "0xf5448fc2beb9324900d08225fe4530ba3bbf654f",
		Exchange: "lista-stable",
		Type:     "lista-stable",
		Reserves: entity.PoolReserves{
			"48615751411650460241085692",
			"11206579925899312237017692",
			"59769030327001165128372730",
		},
		Tokens: []*entity.PoolToken{
			{Address: "0x55d398326f99059ff775485246999027b3197955", Symbol: "USDT", Decimals: 18},
			{Address: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d", Symbol: "USDC", Decimals: 18},
		},
		Extra:       `{"initialA":"500000","futureA":"500000","initialATime":0,"futureATime":0,"swapFee":"100000","adminFee":"2000000000","oraclePrices":[998996790000000000,999662870000000000],"priceDiffThreshold":[50000000000000000,50000000000000000]}`,
		StaticExtra: `{"lpToken":"0xF6136d7e72446C724ecAeef514AE7B2ab4dbb60B","aPrecision":"100","precisionMultipliers":["1","1"],"rates":["1000000000000000000","1000000000000000000"],"isNativeCoins":[false,false]}`,
	})
	require.NoError(t, err)

	testutil.TestCloneState(t, p, poolpkg.CalcAmountOutParams{
		TokenAmountIn: poolpkg.TokenAmount{
			Token:  "0x55d398326f99059ff775485246999027b3197955",
			Amount: new(big.Int).Mul(big.NewInt(1000), new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil)),
		},
		TokenOut: "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d",
	}, nil)
}

// USD1/USDT — 0xd46bfa2b03a8467d3fe1685d8d79eb195cb19675 on Ethereum
// token[0] = USD1 (18-dec), token[1] = USDT (6-dec); balanced ~101k each
// on-chain get_dy(0,1,100e18)=99424932  get_dy(1,0,100e6)=99424931844444344496
var (
	entityUSD1USDT entity.Pool
	_              = json.Unmarshal([]byte(`{
		"address":"0xd46bfa2b03a8467d3fe1685d8d79eb195cb19675",
		"exchange":"lista-stable",
		"type":"lista-stable",
		"reserves":["101194664603798614406","101194664","202389327724267031814"],
		"tokens":[
			{"address":"0x8d0d000ee44948fc98c9b98a4fa4921476f08b0d","symbol":"USD1","decimals":18,"swappable":true},
			{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}
		],
		"extra":"{\"initialA\":\"500000\",\"futureA\":\"500000\",\"initialATime\":0,\"futureATime\":0,\"swapFee\":\"1000000\",\"adminFee\":\"2000000000\",\"oraclePrices\":[999360000000000000,999360000000000000],\"priceDiffThreshold\":[50000000000000000,50000000000000000]}",
		"staticExtra":"{\"lpToken\":\"0xA7ABa0A4603d274D2536AE21cF6AbE7aa7293d9e\",\"aPrecision\":\"100\",\"precisionMultipliers\":[\"1\",\"1000000000000\"],\"rates\":[\"1000000000000000000\",\"1000000000000000000000000000000\"],\"isNativeCoins\":[false,false]}"
	}`), &entityUSD1USDT)
	poolUSD1USDT = lo.Must(NewPoolSimulator(entityUSD1USDT))
)

// USDC/USDT — 0x35c9a4dae1ff05788f24b5b32721d89340cbb636 on Ethereum
// token[0] = USDC (6-dec), token[1] = USDT (6-dec); imbalanced 85k/324k
// on-chain get_dy(0,1,100e6)=100025525  get_dy(1,0,100e6)=99972414
// This pool was falsely rejected before the XP→raw fix in getDyWithoutFee.
var (
	entityUSDCUSDT entity.Pool
	_              = json.Unmarshal([]byte(`{
		"address":"0x35c9a4dae1ff05788f24b5b32721d89340cbb636",
		"exchange":"lista-stable",
		"type":"lista-stable",
		"reserves":["85777228845","324257429898","410020487529258290804668"],
		"tokens":[
			{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},
			{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}
		],
		"extra":"{\"initialA\":\"472311\",\"futureA\":\"1000000\",\"initialATime\":1780282307,\"futureATime\":1780386578,\"swapFee\":\"100000\",\"adminFee\":\"2000000000\",\"oraclePrices\":[999690000000000000,999360000000000000],\"priceDiffThreshold\":[50000000000000000,50000000000000000]}",
		"staticExtra":"{\"lpToken\":\"0x923C67CeD114dd7341e65C0e2be47f3b0927d667\",\"aPrecision\":\"100\",\"precisionMultipliers\":[\"1000000000000\",\"1000000000000\"],\"rates\":[\"1000000000000000000000000000000\",\"1000000000000000000000000000000\"],\"isNativeCoins\":[false,false]}"
	}`), &entityUSDCUSDT)
	poolUSDCUSDT = lo.Must(NewPoolSimulator(entityUSDCUSDT))
)

// WBTC/cbBTC — 0x94e4a9f24a954047adb3ad4434bf1174f6824e16 on Ethereum
// token[0] = WBTC (8-dec), token[1] = cbBTC (8-dec); tiny pool ~$77 each
// checkPriceDiff() already fails on-chain: the $100 test trade (≈157k sat) exceeds the
// entire pool balance (≈122k sat), so any swap must return ErrPriceDiffToken0.
var (
	entityWBTCcbBTC entity.Pool
	_               = json.Unmarshal([]byte(`{
		"address":"0x94e4a9f24a954047adb3ad4434bf1174f6824e16",
		"exchange":"lista-stable",
		"type":"lista-stable",
		"reserves":["122149","122167","2443141304770259"],
		"tokens":[
			{"address":"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599","symbol":"WBTC","decimals":8,"swappable":true},
			{"address":"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf","symbol":"cbBTC","decimals":8,"swappable":true}
		],
		"extra":"{\"initialA\":\"500000\",\"futureA\":\"500000\",\"initialATime\":0,\"futureATime\":0,\"swapFee\":\"1000000\",\"adminFee\":\"2000000000\",\"oraclePrices\":[63550923360000000000000,63578037764550000000000],\"priceDiffThreshold\":[50000000000000000,50000000000000000]}",
		"staticExtra":"{\"lpToken\":\"0x6Ee2d8B77f5559d88057738821951a855EB5F123\",\"aPrecision\":\"100\",\"precisionMultipliers\":[\"10000000000\",\"10000000000\"],\"rates\":[\"10000000000000000000000000000\",\"10000000000000000000000000000\"],\"isNativeCoins\":[false,false]}"
	}`), &entityWBTCcbBTC)
	poolWBTCcbBTC = lo.Must(NewPoolSimulator(entityWBTCcbBTC))
)

// TestPoolSimulator_CalcAmountOut_USD1USDT: 18-dec/6-dec balanced pool, both directions succeed.
// Pool is tiny (~$202 total), so amounts are capped at 1 token to keep the post-swap state
// stable enough for checkPriceDiff's internal $100 test trade to pass.
// on-chain: get_dy(0,1,1e18)=999899  get_dy(1,0,1e6)=999898024016224198
func TestPoolSimulator_CalcAmountOut_USD1USDT(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolUSD1USDT, map[int]map[int]map[string]string{
		0: {1: {"1000000000000000000": "999898"}},
		1: {0: {"1000000": "999898024016224198"}},
	})
}

// TestPoolSimulator_CalcAmountOut_USDCUSDT: 6-dec/6-dec imbalanced pool.
// Both directions must succeed — this pool was falsely rejected before the XP→raw fix.
// on-chain: get_dy(0,1,100e6)=100025525  get_dy(1,0,100e6)=99972414 (simulator rounds to 99972413)
func TestPoolSimulator_CalcAmountOut_USDCUSDT(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolUSDCUSDT, map[int]map[int]map[string]string{
		0: {1: {"100000000": "100025525"}},
		1: {0: {"100000000": "99972413"}},
	})
}

// TestPoolSimulator_CalcAmountOut_WBTCcbBTC: tiny pool where checkPriceDiff legitimately fails.
// The $100 test trade (≈157k sat) exceeds pool balance (≈122k sat), so any swap errors.
func TestPoolSimulator_CalcAmountOut_WBTCcbBTC(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolWBTCcbBTC, map[int]map[int]map[string]string{
		0: {1: {"1000": ErrPriceDiffToken0.Error()}},
		1: {0: {"1000": ErrPriceDiffToken0.Error()}},
	})
}
