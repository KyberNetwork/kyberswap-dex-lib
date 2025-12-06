package rsethl2

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0x291088312150482826b3a37d5a69a4c54daa9118","exchange":"kelp-rseth-l2","type":"kelp-rseth-l2","timestamp":1764906944,"reserves":["100000000000000000000000000","100000000000000000000000000","100000000000000000000000000"],"tokens":[{"address":"0xc1cba3fcea344f92d9239c08c0568f6f2f0ee452","symbol":"wstETH","decimals":18,"swappable":true},{"address":"0x4200000000000000000000000000000000000006","symbol":"WETH","decimals":18,"swappable":true},{"address":"0x291088312150482826b3a37d5a69a4c54daa9118","symbol":"rsETH","decimals":18,"swappable":true}],"extra":"{\"sTOs\":[\"0xefbbf9290cda1c3046211d5464cc52dae46c544c\"],\"sTRates\":[1220611658741126300],\"rsETHRate\":1059840445598399939,\"fee\":0}"}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		0: {
			2: {
				"1000000000000000000": "1151693789202348101", //
			},
		},
		1: {
			2: {
				"1000000000000000000": "943538250642422660", //
			},
		},
	})
}

func TestPoolSimulator_CalcAmountOut_1(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		0: {
			1: {
				"1000000000000000000": "",
			},
		},
		2: {
			1: {
				"1000000000000000000": "",
			},
		},
	})
}

func TestPoolSimulator_CalcAmountOut_2(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		1: {
			0: {
				"1000000000000000000": "",
			},
		},
		2: {
			0: {
				"1000000000000000000": "",
			},
		},
	})
}
