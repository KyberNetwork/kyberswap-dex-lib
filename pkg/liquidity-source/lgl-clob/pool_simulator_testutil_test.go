package lglclob

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// pools p etherlink hanji (block 46540073)
const wxtzUsdcPoolJSON = `{
	"address": "0xc0ba6913a5703e0c8ce0338cf952accfeebcecf9",
	"swapFee": 0.0002,
	"exchange": "hanji",
	"type": "lgl-clob",
	"reserves": ["299115400000000000000000","8336539359"],
	"tokens": [
		{"address":"0xc9b53ab2679f573e480d01e0f49e2b5cfb7a3eab","symbol":"WXTZ","decimals":18,"swappable":true},
		{"address":"0x796ea11fa2dd751ed01b53c372ffdb4aaa8f00f9","symbol":"USDC","decimals":6,"swappable":true}
	],
	"extra": "{\"b\":{\"p\":[\"21769\",\"21767\",\"21758\",\"21750\",\"15000\",\"10000\",\"5000\",\"100\",\"10\"],\"s\":[\"18826\",\"56483\",\"113013\",\"188425\",\"2338\",\"5000\",\"10000\",\"1\",\"500000\"]},\"a\":{\"p\":[\"21816\",\"21818\",\"21827\",\"21835\",\"125000\"],\"s\":[\"45837\",\"229168\",\"916296\",\"1797852\",\"2001\"]}}",
	"staticExtra": "{\"sX\":\"100000000000000000\",\"sY\":\"1\",\"n\":true}"
}`

// pools p etherlink hanji (block 46540079)
const wxtzWethPoolJSON = `{
	"address": "0x0521767cdd2517b3133f0b102ee7d9988413a86a",
	"swapFee": 0.0003,
	"exchange": "hanji",
	"type": "lgl-clob",
	"reserves": ["298923873079000000000000","2738660660507160000"],
	"tokens": [
		{"address":"0xc9b53ab2679f573e480d01e0f49e2b5cfb7a3eab","symbol":"WXTZ","decimals":18,"swappable":true},
		{"address":"0xfc24f770f94edbca6d6f885e12d4317320bcb401","symbol":"WETH","decimals":18,"swappable":true}
	],
	"extra": "{\"b\":{\"p\":[\"1376000\",\"1375860\",\"1375040\",\"1374490\"],\"s\":[\"995152856\",\"2985762353\",\"5975085802\",\"9962461206\"]},\"a\":{\"p\":[\"1378660\",\"1378800\",\"1379630\",\"1380180\"],\"s\":[\"4584823480\",\"22921789743\",\"91631999011\",\"179785260845\"]}}",
	"staticExtra": "{\"sX\":\"1000000000000\",\"sY\":\"100\",\"n\":true}"
}`

func mustNewSim(t *testing.T, rawJSON string) *PoolSimulator {
	t.Helper()
	var ep entity.Pool
	require.NoError(t, json.Unmarshal([]byte(rawJSON), &ep))
	sim, err := NewPoolSimulator(ep)
	require.NoError(t, err)
	return sim
}

// WXTZ/USDC: sfX=1e17 (0.1 WXTZ/share), sfY=1 (1 USDC/share), fee=0.0002
func TestCalcAmountOut_WXTZ_USDC(t *testing.T) {
	t.Parallel()

	sim := mustNewSim(t, wxtzUsdcPoolJSON)

	// token0=WXTZ, token1=USDC
	testutil.TestCalcAmountOut(t, sim, map[int]map[int]map[string]string{
		0: {1: {
			// sell 0.1 WXTZ (1 share at bid 21769): gross=21769, fee=ceil(21769*0.0002)=5, net=21764
			"100000000000000000": "21764",
			// sell 1 WXTZ (10 shares at bid 21769): gross=217690, fee=44, net=217646
			"1000000000000000000": "217646",
			// sell 10 WXTZ (100 shares at bid 21769): gross=2176900, fee=ceil(435.38)=436, net=2176464
			"10000000000000000000": "2176464",
		}},
		1: {0: {
			// buy 0.1 WXTZ (1 share at ask 21816, net cost 21816, fee=ceil(21816*0.0002)=5, amtIn=21821)
			// availableAmtIn = floor(21821/1.0002)=21816, scaledAmtIn=21816, shares=floor(21816/21816)=1
			"21821": "100000000000000000",
			// buy 1 WXTZ (10 shares at ask 21816, net cost 218160, fee=ceil(218160*0.0002)=44, amtIn=218204)
			"218204": "1000000000000000000",
		}},
	})
}

func TestCalcAmountIn_WXTZ_USDC(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, mustNewSim(t, wxtzUsdcPoolJSON))
}

// WXTZ/WETH: sfX=1e12 (1e-6 WXTZ/share), sfY=100 (100 wei WETH/share), fee=0.0003
func TestCalcAmountOut_WXTZ_WETH(t *testing.T) {
	t.Parallel()

	sim := mustNewSim(t, wxtzWethPoolJSON)

	// token0=WXTZ, token1=WETH
	testutil.TestCalcAmountOut(t, sim, map[int]map[int]map[string]string{
		0: {1: {
			// sell 1 WXTZ = 1e18 wei → 1e6 shares at bid 1376000
			// gross = 1e6 * 1376000 = 1376000000000 scaled Y, * sfY=100 → 137600000000000 wei WETH
			// fee = ceil(137600000000000 * 0.0003) = ceil(41280000000000) = wait
			// Actually fee = ceil(137600000000000 * 3e14 / 1e18) = ceil(137600000000000 * 3e-4)
			// = ceil(41280000000) = 41280000001? no: 137600000000000 * 3e-4 = 41280000000
			// fee = 41280000000
			// net = 137600000000000 - 41280000000 = 137558720000000
			"1000000000000000000": "137558720000000",
		}},
		1: {0: {
			// buy 1 WXTZ = 1e18 wei → 1e6 shares at ask 1378660
			// cost = 1e6 * 1378660 scaled Y = 1378660000000 scaled Y * 100 = 137866000000000 wei WETH
			// fee = ceil(137866000000000 * 3e-4) = ceil(41359800000) = 41359800000
			// amtIn = 137866000000000 + 41359800000 = 137907359800000
			// availableAmtIn = floor(137907359800000 / 1.0003) = floor(137866000000000) = 137866000000000 ✓
			"137907359800000": "1000000000000000000",
		}},
	})
}

func TestCalcAmountIn_WXTZ_WETH(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountIn(t, mustNewSim(t, wxtzWethPoolJSON))
}
