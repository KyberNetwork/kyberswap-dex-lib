package bunniv2

import (
	"math/big"
	"reflect"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/bunni-v2/oracle"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TestCloneStateIsolatesObservations pins the isolation contract of CloneState: quoting and
// updating a clone must not touch the source. The observation ring buffer is the field that broke
// it — CloneState copied only the slice header while oracle.set writes into the buffer by index,
// so a clone shared the source's backing array and corrupted a concurrent reader's pool state.
//
// The write happens in updateOracle, reached from BeforeSwap, so CalcAmountOut alone is enough to
// trigger it; UpdateBalance is not required.
func TestCloneStateIsolatesObservations(t *testing.T) {
	t.Parallel()

	var p entity.Pool
	poolData := `{"address":"0xd9f673912e1da331c9e56c5f0dbc7273c0eb684617939a375ec5e227c62d6707","exchange":"uniswap-v4-bunni-v2","type":"uniswap-v4","timestamp":1755504091,"reserves":["5482789212546","15765380488591"],"tokens":[{"address":"0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48","symbol":"USDC","decimals":6,"swappable":true},{"address":"0xdac17f958d2ee523a2206206994597c13d831ec7","symbol":"USDT","decimals":6,"swappable":true}],"extra":"{\"liquidity\":0,\"sqrtPriceX96\":79222893666315702689978882880,\"tickSpacing\":1,\"tick\":-2,\"ticks\":null,\"hX\":\"{\\\"he\\\":\\\"{\\\\\\\"OverrideZeroToOne\\\\\\\":false,\\\\\\\"FeeZeroToOne\\\\\\\":\\\\\\\"0\\\\\\\",\\\\\\\"OverrideOneToZero\\\\\\\":false,\\\\\\\"FeeOneToZero\\\\\\\":\\\\\\\"0\\\\\\\"}\\\",\\\"ha\\\":\\\"0x00ece5a72612258f20eb24573c544f9dd8c5000c\\\",\\\"la\\\":\\\"0x000000000b757686c9596cada54fa28f8c429e0d\\\",\\\"hf\\\":\\\"0\\\",\\\"pmr\\\":[\\\"92841916942359\\\",\\\"71019497118941\\\"],\\\"ls\\\":[1,255,255,246,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],\\\"v\\\":[{\\\"a\\\":\\\"0xe0a80d35bb6618cba260120b279d357978c42bce\\\",\\\"d\\\":6,\\\"rr\\\":\\\"1047514345698780679\\\",\\\"dr\\\":\\\"954640863971094742\\\",\\\"md\\\":\\\"110806078842304\\\",\\\"wr\\\":\\\"954640863971094743\\\",\\\"mw\\\":\\\"4553406902240\\\"},{\\\"a\\\":\\\"0x7c280dbdef569e96c7919251bd2b0edf0734c5a8\\\",\\\"d\\\":6,\\\"rr\\\":\\\"1041844468229355827\\\",\\\"dr\\\":\\\"959836166044561635\\\",\\\"md\\\":\\\"3653094287998\\\",\\\"wr\\\":\\\"959836166044561636\\\",\\\"mw\\\":\\\"2417062853137\\\"}],\\\"aa\\\":{\\\"am\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"sf01\\\":\\\"0\\\",\\\"sf10\\\":\\\"0\\\"},\\\"os\\\":{\\\"i\\\":7,\\\"c\\\":25,\\\"cn\\\":25,\\\"io\\\":{\\\"bt\\\":1755502847,\\\"pt\\\":-7,\\\"tc\\\":-6573732,\\\"i\\\":true}},\\\"cf\\\":{\\\"fr\\\":\\\"0\\\"},\\\"o\\\":[{\\\"bt\\\":1755484463,\\\"pt\\\":-7,\\\"tc\\\":-6443880,\\\"i\\\":true},{\\\"bt\\\":1755486443,\\\"pt\\\":-7,\\\"tc\\\":-6457740,\\\"i\\\":true},{\\\"bt\\\":1755488399,\\\"pt\\\":-7,\\\"tc\\\":-6471432,\\\"i\\\":true},{\\\"bt\\\":1755491291,\\\"pt\\\":-7,\\\"tc\\\":-6491676,\\\"i\\\":true},{\\\"bt\\\":1755493283,\\\"pt\\\":-7,\\\"tc\\\":-6504672,\\\"i\\\":true},{\\\"bt\\\":1755497123,\\\"pt\\\":-8,\\\"tc\\\":-6533664,\\\"i\\\":true},{\\\"bt\\\":1755500087,\\\"pt\\\":-7,\\\"tc\\\":-6554412,\\\"i\\\":true},{\\\"bt\\\":1755502835,\\\"pt\\\":-7,\\\"tc\\\":-6573648,\\\"i\\\":true},{\\\"bt\\\":1755442523,\\\"pt\\\":-6,\\\"tc\\\":-6167496,\\\"i\\\":true},{\\\"bt\\\":1755444851,\\\"pt\\\":-6,\\\"tc\\\":-6181464,\\\"i\\\":true},{\\\"bt\\\":1755446663,\\\"pt\\\":-6,\\\"tc\\\":-6192336,\\\"i\\\":true},{\\\"bt\\\":1755448979,\\\"pt\\\":-6,\\\"tc\\\":-6206232,\\\"i\\\":true},{\\\"bt\\\":1755452039,\\\"pt\\\":-6,\\\"tc\\\":-6224592,\\\"i\\\":true},{\\\"bt\\\":1755453863,\\\"pt\\\":-6,\\\"tc\\\":-6235536,\\\"i\\\":true},{\\\"bt\\\":1755459719,\\\"pt\\\":-6,\\\"tc\\\":-6270672,\\\"i\\\":true},{\\\"bt\\\":1755461519,\\\"pt\\\":-7,\\\"tc\\\":-6283272,\\\"i\\\":true},{\\\"bt\\\":1755463319,\\\"pt\\\":-7,\\\"tc\\\":-6295872,\\\"i\\\":true},{\\\"bt\\\":1755467195,\\\"pt\\\":-7,\\\"tc\\\":-6323004,\\\"i\\\":true},{\\\"bt\\\":1755468995,\\\"pt\\\":-7,\\\"tc\\\":-6335604,\\\"i\\\":true},{\\\"bt\\\":1755471251,\\\"pt\\\":-7,\\\"tc\\\":-6351396,\\\"i\\\":true},{\\\"bt\\\":1755473171,\\\"pt\\\":-7,\\\"tc\\\":-6364836,\\\"i\\\":true},{\\\"bt\\\":1755474971,\\\"pt\\\":-7,\\\"tc\\\":-6377436,\\\"i\\\":true},{\\\"bt\\\":1755477275,\\\"pt\\\":-7,\\\"tc\\\":-6393564,\\\"i\\\":true},{\\\"bt\\\":1755479495,\\\"pt\\\":-7,\\\"tc\\\":-6409104,\\\"i\\\":true},{\\\"bt\\\":1755481835,\\\"pt\\\":-7,\\\"tc\\\":-6425484,\\\"i\\\":true}],\\\"hp\\\":{\\\"fmin\\\":\\\"5\\\",\\\"fmax\\\":\\\"5\\\",\\\"fqm\\\":\\\"0\\\",\\\"ftsa\\\":0,\\\"sfhl\\\":\\\"60\\\",\\\"sfat\\\":120,\\\"vst0\\\":\\\"100\\\",\\\"vst1\\\":\\\"100\\\",\\\"rt\\\":100,\\\"aae\\\":false,\\\"omi\\\":1800},\\\"s0\\\":{\\\"spx96\\\":\\\"79203287901213834391885852055\\\",\\\"t\\\":-7,\\\"lst\\\":1755502847,\\\"lsgt\\\":1755501863},\\\"bs\\\":{\\\"tsa\\\":43200,\\\"lp\\\":[0,255,255,253,0,4,2,250,240,128,29,205,101,0,0,4,11,235,194,0,29,205,101,0,59,154,202,0,0,0,0,0],\\\"hp\\\":\\\"AAAFAAAFAAAAAAAAAAAAADwAeABkAGQAZABkBwgBLAAAAAcIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\\\",\\\"lt\\\":2,\\\"mrtr0\\\":\\\"100000\\\",\\\"trtr0\\\":\\\"200000\\\",\\\"xrtr0\\\":\\\"300000\\\",\\\"mrtr1\\\":\\\"100000\\\",\\\"trtr1\\\":\\\"200000\\\",\\\"xrtr1\\\":\\\"300000\\\",\\\"c0d\\\":6,\\\"c1d\\\":6,\\\"rb0\\\":\\\"1137844034040\\\",\\\"rb1\\\":\\\"3900685510550\\\",\\\"r0\\\":\\\"4344945178506\\\",\\\"r1\\\":\\\"11864694978041\\\",\\\"ib\\\":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,10,247,4,178,47,234]},\\\"vsp\\\":{\\\"i\\\":true,\\\"sp0\\\":\\\"1047510601209677321\\\",\\\"sp1\\\":\\\"1041839960908456872\\\"},\\\"rod\\\":0,\\\"bt\\\":1755504083}\"}","staticExtra":"{\"0x0\":[false,false],\"fee\":0,\"tS\":1,\"hooks\":\"0x000052423c1db6b7ff8641b85a7eefc7b2791888\",\"uR\":\"0x66a9893cc07d91d95644aedd05d03f95e1dba8af\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":23166618}`
	require.NoError(t, json.Unmarshal([]byte(poolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDEthereum)
	require.NoError(t, err)

	before := mustQuote(t, pSim)

	cloned := pSim.CloneState()
	require.NotSame(t, pSim, cloned)
	_ = mustQuote(t, cloned)

	after := mustQuote(t, pSim)
	require.Equal(t, before.TokenAmountOut.Amount.String(), after.TokenAmountOut.Amount.String(),
		"quoting a clone must not move the source's price")

	srcHook := hookOf(t, pSim)
	clonedHook := hookOf(t, cloned)
	require.NotSame(t, srcHook, clonedHook, "the clone must not share the source hook")
	require.False(t, sharesBacking(srcHook.Observations, clonedHook.Observations),
		"the clone's observation ring buffer must not share a backing array with the source")
}

func mustQuote(t *testing.T, sim pool.IPoolSimulator) *pool.CalcAmountOutResult {
	t.Helper()
	res, err := sim.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			Amount: big.NewInt(1000000000),
		},
		TokenOut: "0xdac17f958d2ee523a2206206994597c13d831ec7",
	})
	require.NoError(t, err)
	return res
}

func hookOf(t *testing.T, sim pool.IPoolSimulator) *Hook {
	t.Helper()
	v := reflect.ValueOf(sim).Elem().FieldByName("hook")
	require.True(t, v.IsValid(), "uniswapv4.PoolSimulator must expose a hook field")
	h, ok := reflect.NewAt(v.Type(), v.Addr().UnsafePointer()).Elem().Interface().(*Hook)
	require.True(t, ok, "pool must be built with the bunni-v2 hook")
	return h
}

func sharesBacking(a, b []oracle.Observation) bool {
	if len(a) == 0 || len(b) == 0 {
		return false
	}
	return &a[0] == &b[0]
}
