package unibtc

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

var (
	entityPool entity.Pool
	_          = json.Unmarshal([]byte("{\"address\":\"0x047d41f2544b7f63a8e991af2068a363d210d6da\",\"exchange\":\"bedrock-unibtc\",\"type\":\"bedrock-unibtc\",\"timestamp\":1754559660,\"reserves\":[\"332068154748\",\"499999779514\",\"498897437521\",\"0\"],\"tokens\":[{\"address\":\"0xc96de26018a54d51c097160568752c4e3bd6c364\",\"symbol\":\"FBTC\",\"decimals\":8,\"swappable\":true},{\"address\":\"0xcbb7c0000ab88b473b1f5afd9ef808440eed33bf\",\"symbol\":\"cbBTC\",\"decimals\":8,\"swappable\":true},{\"address\":\"0x2260fac5e5542a773aa44fbcfedf7c193bc2c599\",\"symbol\":\"WBTC\",\"decimals\":8,\"swappable\":true},{\"address\":\"0x004e9c3ef86bc1ca1f0bb5c7662861ee93350568\",\"symbol\":\"uniBTC\",\"decimals\":8,\"swappable\":true}],\"extra\":\"{\\\"paused\\\":false,\\\"tokensPaused\\\":[false,false,false,false],\\\"tokensAllowed\\\":[true,true,true,false],\\\"caps\\\":[500000000000,500000000000,500000000000,0],\\\"exchangeRateBase\\\":10000000000,\\\"supplyFeeder\\\":\\\"0x94c7f81e3b0458daa721ca5e29f6ced05ccce2b3\\\",\\\"tokenUsedCaps\\\":[167931845252,220486,1102562479,null]}\",\"blockNumber\":23088351}"),
		&entityPool)
	poolSim = lo.Must(NewPoolSimulator(entityPool))
)

func TestPoolSimulator_CalcAmountOut(t *testing.T) {
	t.Parallel()
	testutil.TestCalcAmountOut(t, poolSim, map[int]map[int]map[string]string{
		0: {
			1: {"100000000000": ""},
			2: {"100000000000": ""},
			3: {
				"100000000000": "100000000000",
				"332068154747": "332068154747",
				"332068154748": "",
			},
		},
		1: {
			0: {"100000000000": ""},
			2: {"100000000000": ""},
			3: {"100000000000": "100000000000"},
		},
		2: {
			0: {"100000000000": ""},
			1: {"100000000000": ""},
			3: {"100000000000": "100000000000"},
		},
		3: {
			0: {"100000000000": ""},
			1: {"100000000000": ""},
			2: {"100000000000": ""},
		},
	})
}
