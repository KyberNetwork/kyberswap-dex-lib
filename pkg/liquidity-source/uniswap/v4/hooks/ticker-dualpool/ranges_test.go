package tickerdualpool

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
)

func TestRangeConfigurationsMatchSolidity(t *testing.T) {
	raw := []byte(capturedRangeQuotes)
	var rows []map[string]string
	require.NoError(t, json.Unmarshal(raw, &rows))
	require.Len(t, rows, 18)
	for _, row := range rows {
		t.Run(row["config"]+"/"+row["side"]+"/"+row["input"], func(t *testing.T) {
			s := fixtureState()
			config, err := strconv.Atoi(row["config"])
			require.NoError(t, err)
			s.Pool.Buckets = rangeBuckets(config)
			q, err := QuoteExactInput(s, row["side"] == "0", uint256.MustFromDecimal(row["input"]))
			require.NoError(t, err)
			require.Equal(t, row["used"], q.Swap.AmountIn.Dec())
			require.Equal(t, row["net"], q.NetOutput.Dec())
			require.Equal(t, row["fee"], q.HookFee.Dec())
			require.Equal(t, row["tax"], q.CreatorTax.Dec())
			require.Equal(t, row["price"], q.Swap.SqrtPriceX96.Dec())
			require.Equal(t, row["tick"], strconv.Itoa(int(q.Swap.Tick)))
			require.Equal(t, row["after0"], q.Swap.Balance0.Dec())
			require.Equal(t, row["after1"], q.Swap.Balance1.Dec())
		})
	}
}

func TestExhaustedRangesStopAtPriceLimit(t *testing.T) {
	for _, direction := range []bool{true, false} {
		s := fixtureState()
		s.Pool.Buckets = rangeBuckets(2)
		_, err := QuoteExactInput(s, direction, uint256.MustFromDecimal("1000000000000000000000000000000"))
		require.ErrorIs(t, err, dualpool.ErrInsufficientLiquidity)
	}
}

// Captured contract results; preserved from range-quotes.json.
const capturedRangeQuotes = `[
{"config":"0","side":"0","input":"1001","used":"1001","net":"984","fee":"9","tax":"4","price":"79228162514264337593533139165","tick":"-1","after0":"1000000000000000000000994","after1":"999999999999999999998998"},
{"config":"0","side":"0","input":"100000000000000000000000","used":"100000000000000000000000","net":"96979652778115454959179","fee":"984565002823507156945","tax":"492282501411753578472","price":"78161599248054059552596165528","tick":"-272","after0":"1099999999999999999999990","after1":"901543499717649284305399"},
{"config":"0","side":"0","input":"900000000000000000000000","used":"900000000000000000000000","net":"769394391265347053899834","fee":"7811110571221797501521","tax":"3905555285610898750760","price":"66558260224648763234167231838","tick":"-3486","after0":"1899999999999999999999987","after1":"218888942877820249847879"},
{"config":"0","side":"1","input":"1001","used":"1001","net":"984","fee":"9","tax":"4","price":"79228162514264337593554761507","tick":"0","after0":"999999999999999999998998","after1":"1000000000000000000000994"},
{"config":"0","side":"1","input":"100000000000000000000000","used":"100000000000000000000000","net":"96979652778115454959179","fee":"984565002823507156945","tax":"492282501411753578472","price":"80309279694567634368040595647","tick":"271","after0":"901543499717649284305399","after1":"1099999999999999999999990"},
{"config":"0","side":"1","input":"900000000000000000000000","used":"900000000000000000000000","net":"769394391265347053899834","fee":"7811110571221797501521","tax":"3905555285610898750760","price":"94309883013769924033972339152","tick":"3485","after0":"218888942877820249847879","after1":"1899999999999999999999987"},
{"config":"1","side":"0","input":"1001","used":"1001","net":"984","fee":"9","tax":"4","price":"79228162514264337593534310119","tick":"-1","after0":"1000000000000000000000998","after1":"999999999999999999999000"},
{"config":"1","side":"0","input":"100000000000000000000000","used":"100000000000000000000000","net":"97121261002939391028590","fee":"986002649776034426686","tax":"493001324888017213343","price":"78275729635221361550724261209","tick":"-242","after0":"1099999999999999999999997","after1":"901399735022396557331378"},
{"config":"1","side":"0","input":"900000000000000000000000","used":"900000000000000000000000","net":"663439066054635801801981","fee":"6735421990402393926923","tax":"3367710995201196963461","price":"50139808328607636078131296296","tick":"-9151","after0":"1899999999999999999999994","after1":"326457800959760607307631"},
{"config":"1","side":"1","input":"1001","used":"1001","net":"984","fee":"9","tax":"4","price":"79228162514264337593553590553","tick":"0","after0":"999999999999999999999000","after1":"1000000000000000000000998"},
{"config":"1","side":"1","input":"100000000000000000000000","used":"100000000000000000000000","net":"97121261002939391028590","fee":"986002649776034426686","tax":"493001324888017213343","price":"80192184277796918137511661644","tick":"241","after0":"901399735022396557331378","after1":"1099999999999999999999997"},
{"config":"1","side":"1","input":"900000000000000000000000","used":"900000000000000000000000","net":"663439066054635801801983","fee":"6735421990402393926923","tax":"3367710995201196963461","price":"125191977086303184837846337952","tick":"9150","after0":"326457800959760607307630","after1":"1899999999999999999999994"},
{"config":"2","side":"0","input":"1001","used":"1001","net":"984","fee":"9","tax":"4","price":"79228162514264337593540094249","tick":"-1","after0":"1000000000000000000000998","after1":"999999999999999999999000"},
{"config":"2","side":"0","input":"100000000000000000000000","used":"100000000000000000000000","net":"97826870154779851415418","fee":"993166194464770065131","tax":"496583097232385032565","price":"78844421501731724578844255705","tick":"-98","after0":"1099999999999999999999997","after1":"900683380553522993486884"},
{"config":"2","side":"0","input":"900000000000000000000000","used":"900000000000000000000000","net":"847599111730193914358413","fee":"8605067124164405221912","tax":"4302533562082202610956","price":"75903324043581691840541567836","tick":"-858","after0":"1899999999999999999999997","after1":"139493287583559477808718"},
{"config":"2","side":"1","input":"1001","used":"1001","net":"984","fee":"9","tax":"4","price":"79228162514264337593547806423","tick":"0","after0":"999999999999999999999000","after1":"1000000000000000000000998"},
{"config":"2","side":"1","input":"100000000000000000000000","used":"100000000000000000000000","net":"97826870154779851415418","fee":"993166194464770065131","tax":"496583097232385032565","price":"79613771219677369811131034859","tick":"97","after0":"900683380553522993486884","after1":"1099999999999999999999997"},
{"config":"2","side":"1","input":"900000000000000000000000","used":"900000000000000000000000","net":"847599111730193914358413","fee":"8605067124164405221912","tax":"4302533562082202610956","price":"82698640862981627551827711046","tick":"857","after0":"139493287583559477808718","after1":"1899999999999999999999997"}
]`
