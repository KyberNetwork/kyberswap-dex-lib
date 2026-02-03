package stabull

import (
	"encoding/json"
	"testing"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

const (
	multicall3Address = "0xcA11bde05977b3631167028862bE2a173976CA11"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte(`{"address":"0x8a908ae045e611307755a91f4d6ecd04ed31eb1b","swapFee":0.00015,"exchange":"stabull","type":"stabull","timestamp":1770137894,"reserves":["81042678433308405213086","7408220122"],"tokens":[{"address":"0xe9185ee218cae427af7b9764a011bb89fea761b4","symbol":"BRZ","decimals":18,"swappable":true},{"address":"0x833589fcd6edb6e08f4c7c32d4f71b54bda02913","symbol":"USDC","decimals":6,"swappable":true}],"extra":"{\"c\":{\"a\":\"9223372036854775826\",\"b\":\"6456360425798343084\",\"d\":\"9223372036854775826\",\"e\":\"2767011611056451\",\"l\":\"18446744073709551634\"},\"o\":[\"19154519\",\"99973367\"],\"r\":\"191596217820692184\"}","staticExtra":"{\"a\":[\"0x8ba5bddc1cd6d1a0c757982b2af3eb6db53903e0\",\"0x53b105e1d48a76cdb955d037f042c830d14d82ab\"]}"}`),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		0: {
			0: {
				"1000000000": "invalid token",
			},
			1: {
				"53866961516229112":      "10236",
				"255386696151622911293":  "48330208",
				"5255386696151622911293": "920970581",
			},
		},
		1: {
			0: {
				"1":         "invalid amount",
				"48330208":  "253170343540645724922",
				"920970581": "4807067524700350122287",
			},
			1: {
				"1000000000": "invalid token",
			},
		},
	})
}
