package wildcat

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0x4ec0cb2ef6c438f1c1634212377abdc9f2b0b20d","exchange":"wildcat","type":"wildcat","timestamp":1765885780,"reserves":["118004000000000000000","999930756215080"],"tokens":[{"address":"0x6a5e34ea281b25a860517077a7a942e95ec31154","symbol":"ETH","decimals":18,"swappable":true},{"address":"0xb2b18b723f6d0df5be06e245f6eebf8a43d85151","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"rates\":[332665881250712632291671,355429528951],\"isNative\":[false,false]}"}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		1: {
			0: {
				"3005810": "997940729029576",
			},
		},
		0: {
			1: {
				"1000000000000000": "3005810",
			},
		},
	})
}
