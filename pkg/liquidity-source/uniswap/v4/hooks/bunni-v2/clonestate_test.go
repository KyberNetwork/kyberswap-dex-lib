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

// TestCloneStateIsolatesObservations pins the isolation of the observation ring buffer.
//
// CloneState copied only the slice header and shared the *ObservationStorage wrapping the same
// slice, while oracle.set writes into that buffer by index — so a clone wrote into the source's
// backing array. The write happens in updateOracle, reached from BeforeSwap, so CalcAmountOut is
// enough to trigger it; UpdateBalance never touches Observations.
//
// Two conditions are load-bearing and the test would be vacuous without either:
//   - the fixture uses the OracleUniGeo LDF, the variant that runs the oracle at all;
//   - the source is NEVER quoted, so the clone is the first writer after the state refresh.
//     Quoting the source first consumes its writeObservationOnce and pushes the clone's write into
//     oracle.Write's "interval not elapsed" branch, which lands on a value field instead.
func TestCloneStateIsolatesObservations(t *testing.T) {
	t.Parallel()

	var p entity.Pool
	poolData := `{"address":"0x54ff1fd1d62f3bc6224082ecfdb3190a34e8428611b058ade19ce6c083cb608b","exchange":"uniswap-v4-bunni-v2","type":"uniswap-v4","timestamp":1755522881,"reserves":["4765811330936679837190504","1418492956628879905908488"],"tokens":[{"address":"0x35d8949372d46b7a3d5a56006ae77b215fc69bc0","symbol":"USD0++","decimals":18,"swappable":true},{"address":"0x73a15fed60bf67631dc6cd7bc5b6e8da8190acf5","symbol":"USD0","decimals":18,"swappable":true}],"extra":"{\"liquidity\":0,\"sqrtPriceX96\":76190265766238121671454010019,\"tickSpacing\":5,\"tick\":-783,\"ticks\":null,\"hX\":\"{\\\"he\\\":\\\"{\\\\\\\"OverrideZeroToOne\\\\\\\":false,\\\\\\\"FeeZeroToOne\\\\\\\":\\\\\\\"0\\\\\\\",\\\\\\\"OverrideOneToZero\\\\\\\":false,\\\\\\\"FeeOneToZero\\\\\\\":\\\\\\\"0\\\\\\\"}\\\",\\\"ha\\\":\\\"0x0000e819b8a536cf8e5d70b9c49256911033000c\\\",\\\"la\\\":\\\"0x00000000b5cd5d1e09a5c1fb166d26d1cef0c33c\\\",\\\"hf\\\":\\\"0\\\",\\\"pmr\\\":[\\\"4828638527591651254311369\\\",\\\"1466493084648341862790220\\\"],\\\"ls\\\":[1,255,252,189,3,1,1,0,0,0,0,0,5,219,240,96,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],\\\"v\\\":[{\\\"a\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"d\\\":0,\\\"rr\\\":\\\"0\\\",\\\"dr\\\":\\\"0\\\",\\\"md\\\":\\\"0\\\",\\\"wr\\\":\\\"0\\\",\\\"mw\\\":\\\"0\\\"},{\\\"a\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"d\\\":0,\\\"rr\\\":\\\"0\\\",\\\"dr\\\":\\\"0\\\",\\\"md\\\":\\\"0\\\",\\\"wr\\\":\\\"0\\\",\\\"mw\\\":\\\"0\\\"}],\\\"aa\\\":{\\\"am\\\":\\\"0x0000000000000000000000000000000000000000\\\",\\\"sf01\\\":\\\"0\\\",\\\"sf10\\\":\\\"0\\\"},\\\"os\\\":{\\\"i\\\":1,\\\"c\\\":2,\\\"cn\\\":2,\\\"io\\\":{\\\"bt\\\":1755511415,\\\"pt\\\":-767,\\\"tc\\\":-3177859800,\\\"i\\\":true}},\\\"cf\\\":{\\\"fr\\\":\\\"0\\\"},\\\"o\\\":[{\\\"bt\\\":1755509495,\\\"pt\\\":-754,\\\"tc\\\":-3176393412,\\\"i\\\":true},{\\\"bt\\\":1755511415,\\\"pt\\\":-767,\\\"tc\\\":-3177859800,\\\"i\\\":true}],\\\"hp\\\":{\\\"fmin\\\":\\\"400\\\",\\\"fmax\\\":\\\"400\\\",\\\"fqm\\\":\\\"0\\\",\\\"ftsa\\\":0,\\\"sfhl\\\":\\\"1\\\",\\\"sfat\\\":0,\\\"vst0\\\":\\\"1\\\",\\\"vst1\\\":\\\"1\\\",\\\"rt\\\":1000,\\\"aae\\\":false,\\\"omi\\\":1800},\\\"s0\\\":{\\\"spx96\\\":\\\"76274500331769221081611528942\\\",\\\"t\\\":-760,\\\"lst\\\":1755511415,\\\"lsgt\\\":0},\\\"bs\\\":{\\\"tsa\\\":0,\\\"lp\\\":[3,1,1,0,0,0,0,0,5,219,240,96,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0],\\\"hp\\\":\\\"AAGQAAGQAAAAAAAAAAAAAAEAAAABAAED6APoBwgBLAAAAAcIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA==\\\",\\\"lt\\\":2,\\\"mrtr0\\\":\\\"0\\\",\\\"trtr0\\\":\\\"0\\\",\\\"xrtr0\\\":\\\"0\\\",\\\"mrtr1\\\":\\\"0\\\",\\\"trtr1\\\":\\\"0\\\",\\\"xrtr1\\\":\\\"0\\\",\\\"c0d\\\":18,\\\"c1d\\\":18,\\\"rb0\\\":\\\"4765811330936679837190504\\\",\\\"rb1\\\":\\\"1418492956628879905908488\\\",\\\"r0\\\":\\\"0\\\",\\\"r1\\\":\\\"0\\\",\\\"ib\\\":[128,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]},\\\"vsp\\\":{\\\"i\\\":false,\\\"sp0\\\":\\\"0\\\",\\\"sp1\\\":\\\"0\\\"},\\\"rod\\\":0,\\\"bt\\\":0,\\\"oug\\\":{\\\"BondLtStablecoin\\\":true,\\\"FloorPrice\\\":\\\"920000000000000000\\\",\\\"LdfParamOverride\\\":{\\\"Overridden\\\":false,\\\"LdfParams\\\":[0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0]}},\\\"po\\\":\\\"0x35d8949372d46b7a3d5a56006ae77b215fc69bc0\\\"}\"}","staticExtra":"{\"0x0\":[false,false],\"fee\":0,\"tS\":5,\"hooks\":\"0x000052423c1db6b7ff8641b85a7eefc7b2791888\",\"uR\":\"0x66a9893cc07d91d95644aedd05d03f95e1dba8af\",\"pm2\":\"0x000000000022d473030f116ddee9f6b43ac78ba3\",\"mc3\":\"0xca11bde05977b3631167028862be2a173976ca11\"}","blockNumber":23168171}`
	require.NoError(t, json.Unmarshal([]byte(poolData), &p))

	pSim, err := uniswapv4.NewPoolSimulator(p, valueobject.ChainIDEthereum)
	require.NoError(t, err)

	srcHook := hookOf(t, pSim)
	require.NotEmpty(t, srcHook.Observations, "fixture must carry observations, else this proves nothing")
	before := append([]oracle.Observation(nil), srcHook.Observations...)

	cloned := pSim.CloneState()
	require.False(t, sharesBacking(srcHook.Observations, hookOf(t, cloned).Observations),
		"the clone's observation ring buffer must not share a backing array with the source")

	_, err = cloned.CalcAmountOut(pool.CalcAmountOutParams{
		TokenAmountIn: pool.TokenAmount{
			Token:  "0x35d8949372d46b7a3d5a56006ae77b215fc69bc0",
			Amount: big.NewInt(1e18),
		},
		TokenOut: "0x73a15fed60bf67631dc6cd7bc5b6e8da8190acf5",
	})
	require.NoError(t, err)

	require.Equal(t, before, srcHook.Observations,
		"quoting a clone must not write into the source's observation ring buffer")
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
