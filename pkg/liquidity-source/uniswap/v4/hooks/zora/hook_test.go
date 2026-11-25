package zora

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	weth = "0x4200000000000000000000000000000000000006"
	udof = "0xf04515ce02e4a05a14d6cb3d09eaa5d9136bcdd6"
)

// zoraPool is the Base v2.6.0 pool 0xdd9ba443...755c (PoolKey.fee = DYNAMIC_FEE_FLAG), content coin UDOF.
const zoraPool = `{"address":"0xdd9ba443a43973fafed672f6fadedaa107d7eca218fe7cebdfe22b2f371e755c","exchange":"uniswap-v4-zora","type":"uniswap-v4","timestamp":1795044669,"reserves":["0","990015160410896199992737792"],"tokens":[{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true},{"address":"0xf04515ce02e4a05a14d6cb3d09eaa5d9136bcdd6","symbol":"UDOF","decimals":18,"swappable":true}],"extra":"{\"liquidity\":0,\"sqrtPriceX96\":1461446703485210103287273052203988822378723970341,\"tickSpacing\":200,\"tick\":887271,\"ticks\":[{\"index\":-887200,\"liquidityGross\":4356372415375080719360637,\"liquidityNet\":4356372415375080719360637},{\"index\":90200,\"liquidityGross\":8638371161949064091980545,\"liquidityNet\":-74373668801097346740729},{\"index\":93600,\"liquidityGross\":1603956680250374596173444,\"liquidityNet\":-1603956680250374596173444},{\"index\":96800,\"liquidityGross\":760221154013261484657453,\"liquidityNet\":-760221154013261484657453},{\"index\":100000,\"liquidityGross\":470055302653244922569709,\"liquidityNet\":-470055302653244922569709},{\"index\":103200,\"liquidityGross\":324644182922125296664932,\"liquidityNet\":-324644182922125296664932},{\"index\":106600,\"liquidityGross\":233944596160785272433548,\"liquidityNet\":-233944596160785272433548},{\"index\":109800,\"liquidityGross\":178573698707037864418902,\"liquidityNet\":-178573698707037864418902},{\"index\":113000,\"liquidityGross\":139756499110486249640024,\"liquidityNet\":-139756499110486249640024},{\"index\":116200,\"liquidityGross\":111351935579136153012420,\"liquidityNet\":-111351935579136153012420},{\"index\":119600,\"liquidityGross\":88746579249499340065745,\"liquidityNet\":-88746579249499340065745},{\"index\":122800,\"liquidityGross\":72427873646667689307600,\"liquidityNet\":-72427873646667689307600},{\"index\":126000,\"liquidityGross\":59573014917164670254756,\"liquidityNet\":-59573014917164670254756},{\"index\":129200,\"liquidityGross\":49304027671975339624797,\"liquidityNet\":-49304027671975339624797},{\"index\":132600,\"liquidityGross\":40544949099228870236655,\"liquidityNet\":-40544949099228870236655},{\"index\":135800,\"liquidityGross\":33867307829314829277693,\"liquidityNet\":-33867307829314829277693},{\"index\":139000,\"liquidityGross\":28381873415998887937346,\"liquidityNet\":-28381873415998887937346},{\"index\":142200,\"liquidityGross\":23848907235623359847802,\"liquidityNet\":-23848907235623359847802},{\"index\":145600,\"liquidityGross\":19871380653025112272443,\"liquidityNet\":-19871380653025112272443},{\"index\":148800,\"liquidityGross\":16767655023983705380951,\"liquidityNet\":-16767655023983705380951},{\"index\":152000,\"liquidityGross\":14170367465349341679984,\"liquidityNet\":-14170367465349341679984},{\"index\":155200,\"liquidityGross\":420315188482431365502813,\"liquidityNet\":396333666543030591175405},{\"index\":155600,\"liquidityGross\":111079919332959531134589,\"liquidityNet\":-111079919332959531134589},{\"index\":155800,\"liquidityGross\":73682420806129753767322,\"liquidityNet\":-73682420806129753767322},{\"index\":156000,\"liquidityGross\":54984606347746505062105,\"liquidityNet\":-54984606347746505062105},{\"index\":156400,\"liquidityGross\":36288661312501357475231,\"liquidityNet\":-36288661312501357475231},{\"index\":156600,\"liquidityGross\":30947763805547368046555,\"liquidityNet\":-30947763805547368046555},{\"index\":156800,\"liquidityGross\":27010165584688239127564,\"liquidityNet\":-26874950197219530212198},{\"index\":157000,\"liquidityGross\":70688260683201306472,\"liquidityNet\":-64527126785507608894},{\"index\":157200,\"liquidityGross\":21346147076528782588010,\"liquidityNet\":-21332795450079797754678},{\"index\":157400,\"liquidityGross\":19304969719214621994146,\"liquidityNet\":-19304969719214621994146},{\"index\":157600,\"liquidityGross\":17680618099344880665101,\"liquidityNet\":-17519764234609572099857},{\"index\":157800,\"liquidityGross\":26857207555501768893891,\"liquidityNet\":-5630890804373097473177},{\"index\":158000,\"liquidityGross\":111625856337559403960,\"liquidityNet\":-70433839741215115576},{\"index\":158200,\"liquidityGross\":3375636465064584013331,\"liquidityNet\":-3330114722013750796847},{\"index\":158400,\"liquidityGross\":2259262697122148024330,\"liquidityNet\":-2207047755856186444910},{\"index\":158600,\"liquidityGross\":1689087413459738708433,\"liquidityNet\":-1662087093772382856529},{\"index\":158800,\"liquidityGross\":1345176879730811492195,\"liquidityNet\":-1307730385458300762283},{\"index\":159000,\"liquidityGross\":1306353714348034965553,\"liquidityNet\":-908335363345987609879},{\"index\":159200,\"liquidityGross\":1129649004666974912028,\"liquidityNet\":-1125169426086049336638},{\"index\":159400,\"liquidityGross\":4405571492267534843,\"liquidityNet\":-74007088658040547},{\"index\":159600,\"liquidityGross\":2165782201804747148,\"liquidityNet\":-2165782201804747148}],\"sB\":59271471}","staticExtra":"{\"0x0\":[false,false],\"fee\":8388608,\"tS\":200,\"hooks\":\"0x0469a4bd3724dc86c9542f4694c976da13c450c0\",\"uR\":\"0x6ff5693b99212da76ad316178a184ab56d299b43\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":59271471}`

// TestDynamicFeeMatchesChain pins the quote to V4Quoter.quoteExactInputSingle on Base at block
// 59271471 (selling 0.001 WETH). The pool's slot0.lpFee is 0 and the hook's beforeSwap overrides
// it with 1%; without that the quote is ~1% too high.
func TestDynamicFeeMatchesChain(t *testing.T) {
	t.Parallel()
	var ent entity.Pool
	require.NoError(t, json.Unmarshal([]byte(zoraPool), &ent))
	sim, err := uniswapv4.NewPoolSimulator(ent, valueobject.ChainIDBase)
	require.NoError(t, err)

	res, err := testutil.CalcAmountOutNative(sim,
		pool.TokenAmount{Token: weth, Amount: big.NewInt(1e15)},
		udof, nil)

	require.NoError(t, err)
	require.Equal(t, "8097196988910049141775", res.TokenAmountOut.Amount.String())
}

func TestFee(t *testing.T) {
	t.Parallel()
	const created = 1_000_000
	h := func(addr string, createdAt int64, trend bool) *Hook {
		return &Hook{CreatedAt: createdAt, Trend: trend, hookAddress: common.HexToAddress(addr)}
	}
	v26, v24 := trendHook.Hex(), "0xF6d0A13609bb5779Bc5D639F2bA3Bfda83D4D0C0"
	for name, tc := range map[string]struct {
		hook *Hook
		now  int64
		want uniswapv4.FeeAmount
	}{
		"content after launch": {h(v26, created, false), created + 10, 10_000},
		"trend v2.6.0":         {h(v26, created, true), created + 10, 100},
		"trend v2.4.0":         {h(v24, created, true), created + 10, 10_000},
		"launch start":         {h(v26, created, false), created, 990_000},
		"launch mid":           {h(v26, created, false), created + 5, 500_000},
		"trend launch":         {h(v26, created, true), created + 9, 108_000},
		"legacy or untracked":  {h(v26, 0, true), created, 10_000},
	} {
		require.Equal(t, tc.want, tc.hook.fee(tc.now), name)
	}

	got, err := h(v26, 0, false).BeforeSwap(nil)
	require.NoError(t, err)
	require.Equal(t, uniswapv4.OverrideFeeFlag|10_000, got.SwapFee)
}

// TestTrack reads the coin's creation info from Base (UDOF is a content coin created at 1794422410).
func TestTrack(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("needs a Base RPC")
	}
	rpcClient := ethrpc.New("https://base-rpc.publicnode.com").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	var ent entity.Pool
	require.NoError(t, json.Unmarshal([]byte(zoraPool), &ent))
	param := &uniswapv4.HookParam{RpcClient: rpcClient, Pool: &ent, HookAddress: trendHook}

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	hX, err := h.Track(context.Background(), param)
	require.NoError(t, err)
	require.JSONEq(t, `{"c":1794422410}`, string(hX))
}
