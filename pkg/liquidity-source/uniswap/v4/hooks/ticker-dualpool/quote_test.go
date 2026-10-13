package tickerdualpool

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
)

func fixtureState() State {
	return State{
		Pool: dualpool.Extra{Live: true, Balance0: uint256.MustFromDecimal("1000000000000000000000000"),
			Balance1:     uint256.MustFromDecimal("1000000000000000000000000"),
			SqrtPriceX96: uint256.MustFromDecimal("79228162514264337593543950336"),
			LpFee:        2000, Buckets: []dualpool.Bucket{{TickLower: -10000, TickUpper: 10000, WeightBps: 10000}}},
		Total0:      uint256.MustFromDecimal("1000000000000000000000000"),
		Total1:      uint256.MustFromDecimal("1000000000000000000000000"),
		TickSpacing: 10, Registered: true, HookFeeBps: 100, CreatorTaxBps: 50,
	}
}

// Values come from an executed Ticker hook, not from another Go pricing function.
func TestQuotesMatchTickerSolidity(t *testing.T) {
	raw := []byte(capturedQuotes)
	var fixtures []map[string]string
	require.NoError(t, json.Unmarshal(raw, &fixtures))
	for _, f := range fixtures {
		t.Run(f["side"]+"/"+f["input"], func(t *testing.T) {
			s := fixtureState()
			before, _ := json.Marshal(s)
			q, err := QuoteExactInput(s, f["side"] == "0", uint256.MustFromDecimal(f["input"]))
			require.NoError(t, err)
			require.Equal(t, f["net"], q.NetOutput.Dec())
			require.Equal(t, f["after0"], q.Swap.Balance0.Dec())
			require.Equal(t, f["after1"], q.Swap.Balance1.Dec())
			require.Equal(t, f["fee"], q.HookFee.Dec())
			require.Equal(t, f["tax"], q.CreatorTax.Dec())
			require.Equal(t, f["price"], q.Swap.SqrtPriceX96.Dec())
			tick, err := strconv.Atoi(f["tick"])
			require.NoError(t, err)
			require.Equal(t, int32(tick), q.Swap.Tick)
			after, _ := json.Marshal(s)
			require.Equal(t, string(before), string(after), "quoting must not mutate pool state")
		})
	}
}

func TestFloorRejectsShortageAndUnbackedClaims(t *testing.T) {
	for _, which := range []string{"partial", "zero", "unbacked0", "unbacked1"} {
		t.Run(which, func(t *testing.T) {
			s := fixtureState()
			switch which {
			case "partial":
				s.Pool.Balance1.SubUint64(s.Pool.Balance1, 1)
			case "zero":
				s.Pool.Balance1.Clear()
			case "unbacked0":
				s.Unbacked0 = uint256.NewInt(1)
			case "unbacked1":
				s.Unbacked1 = uint256.NewInt(1)
			}
			_, err := QuoteExactInput(s, true, uint256.NewInt(1001))
			require.ErrorIs(t, err, ErrVaultIlliquid)
		})
	}
}

func TestQuotesMatchFreshMonadAaveDeployment(t *testing.T) {
	raw := []byte(capturedMonadQuotes)
	var fixtures []map[string]string
	require.NoError(t, json.Unmarshal(raw, &fixtures))
	for _, f := range fixtures {
		t.Run(f["direction"], func(t *testing.T) {
			s := fixtureState()
			s.Pool.Balance0 = uint256.MustFromDecimal(f["before0"])
			s.Pool.Balance1 = uint256.MustFromDecimal(f["before1"])
			s.Total0 = s.Pool.Balance0.Clone()
			s.Total1 = s.Pool.Balance1.Clone()
			s.Pool.SqrtPriceX96 = uint256.MustFromDecimal(f["initialPrice"])
			tick, err := strconv.Atoi(f["initialTick"])
			require.NoError(t, err)
			s.Pool.Tick = int32(tick)
			s.Pool.Buckets = []dualpool.Bucket{{TickLower: 312380, TickUpper: 332380, WeightBps: 10000}}
			q, err := QuoteExactInput(s, f["direction"] == "true", uint256.MustFromDecimal(f["input"]))
			require.NoError(t, err)
			require.Equal(t, f["net"], q.NetOutput.Dec())
			require.Equal(t, f["price"], q.Swap.SqrtPriceX96.Dec())
			tick, err = strconv.Atoi(f["tick"])
			require.NoError(t, err)
			require.Equal(t, int32(tick), q.Swap.Tick)
		})
	}
}

func TestReverseQuoteMatchesExhaustiveSmallInputs(t *testing.T) {
	for _, direction := range []bool{true, false} {
		s := fixtureState()
		// Around both separate fee boundaries, net output is not monotone.
		for _, target := range []uint64{1, 97, 98, 99, 100, 196, 197, 198, 199, 200, 201, 395} {
			var expected uint64
			for input := uint64(1); input < 1000; input++ {
				q, err := QuoteExactInput(s, direction, uint256.NewInt(input))
				if err == nil && q.NetOutput.CmpUint64(target) >= 0 {
					expected = input
					break
				}
			}
			require.NotZero(t, expected)
			q, err := QuoteRequiredInput(s, direction, uint256.NewInt(target))
			require.NoError(t, err)
			require.Equal(t, expected, q.Swap.AmountIn.Uint64(), "target %d", target)
			require.GreaterOrEqual(t, q.NetOutput.Uint64(), target)
		}
	}
}

// Unlike the fork snapshot test, each quote here uses only the balances and price
// computed by the PREVIOUS Go quote. No intermediate Solidity state is injected.
func TestSequentialQuotesMatchSolidity(t *testing.T) {
	raw := []byte(capturedSequential)
	var fixtures []map[string]string
	require.NoError(t, json.Unmarshal(raw, &fixtures))
	s := fixtureState()
	for _, f := range fixtures {
		q, err := QuoteExactInput(s, f["direction"] == "true", uint256.MustFromDecimal(f["input"]))
		require.NoError(t, err)
		require.Equal(t, f["net"], q.NetOutput.Dec())
		require.Equal(t, f["price"], q.Swap.SqrtPriceX96.Dec())
		require.Equal(t, f["after0"], q.Swap.Balance0.Dec())
		require.Equal(t, f["after1"], q.Swap.Balance1.Dec())
		s.Pool.Balance0 = q.Swap.Balance0.Clone()
		s.Pool.Balance1 = q.Swap.Balance1.Clone()
		s.Total0 = s.Pool.Balance0.Clone()
		s.Total1 = s.Pool.Balance1.Clone()
		s.Pool.SqrtPriceX96 = q.Swap.SqrtPriceX96.Clone()
		s.Pool.Tick = q.Swap.Tick
	}
}

// Captured contract results; preserved from monad-quotes.json.
const capturedMonadQuotes = `[
{"direction":"true","input":"100000000","before0":"9999999999","before1":"1000000000000000000000000","initialPrice":"792359631301295627757723824548136721","initialTick":"322380","net":"9793771323500773836399","price":"789259855873233267524883977085169160","tick":"322301"},
{"direction":"false","input":"1000000000000000000000","before0":"10099999996","before1":"990057084950760635699086","initialPrice":"789259855873233267524883977085169160","initialTick":"322301","net":"9901813","price":"789570989566107662032193314880092797","tick":"322309"}
]`

// Captured contract results; preserved from quotes.json.
const capturedQuotes = `[
{"side":"0","input":"1001","before0":"1000000000000000000000000","before1":"1000000000000000000000000","net":"984","fee":"9","tax":"4","price":"79228162514264337593512840030","tick":"-1","after0":"1000000000000000000000999","after1":"999999999999999999999001"},
{"side":"0","input":"1000000000000000000","before0":"1000000000000000000000000","before1":"1000000000000000000000000","net":"983029613996445648","fee":"9979996081182189","tax":"4989998040591094","price":"79228131403970312680411917320","tick":"-1","after0":"1000000999999999999999998","after1":"999999002000391881781068"},
{"side":"0","input":"100000000000000000000000","before0":"1000000000000000000000000","before1":"1000000000000000000000000","net":"94588807137724631302192","fee":"960292458251011485301","tax":"480146229125505742650","price":"76234676296125791142139077264","tick":"-771","after0":"1099999999999999999999998","after1":"903970754174898851469856"},
{"side":"0","input":"400000000000000000000000","before0":"1000000000000000000000000","before1":"1000000000000000000000000","net":"339835144667195627262804","fee":"3450102991545133271703","tax":"1725051495772566635851","price":"68473276679631106898443301960","tick":"-2918","after0":"1399999999999999999999998","after1":"654989700845486672829640"},
{"side":"1","input":"1001","before0":"1000000000000000000000000","before1":"1000000000000000000000000","net":"984","fee":"9","tax":"4","price":"79228162514264337593575060642","tick":"0","after0":"999999999999999999999001","after1":"1000000000000000000000999"},
{"side":"1","input":"1000000000000000000","before0":"1000000000000000000000000","before1":"1000000000000000000000000","net":"983029613996445648","fee":"9979996081182189","tax":"4989998040591094","price":"79228193624570578500883667574","tick":"0","after0":"999999002000391881781068","after1":"1000000999999999999999998"},
{"side":"1","input":"100000000000000000000000","before0":"1000000000000000000000000","before1":"1000000000000000000000000","net":"94588807137724631302192","fee":"960292458251011485301","tax":"480146229125505742650","price":"82339193138355071565267811747","tick":"770","after0":"903970754174898851469856","after1":"1099999999999999999999998"},
{"side":"1","input":"400000000000000000000000","before0":"1000000000000000000000000","before1":"1000000000000000000000000","net":"339835144667195627262805","fee":"3450102991545133271703","tax":"1725051495772566635851","price":"91672285010627273480439347422","tick":"2917","after0":"654989700845486672829640","after1":"1399999999999999999999998"}
]`

// Captured contract results; preserved from sequential.json.
const capturedSequential = `[
{"direction":"true","input":"50000000000000000000000","net":"48205072294339171296509","price":"77702599142401861968052161033","tick":"-389","after0":"1049999999999999999999998","after1":"951060840310315562135524"},
{"direction":"false","input":"100000000000000000000000","net":"98266565977864051109609","price":"80813629766492595939775986809","tick":"396","after0":"950236988854960354203440","after1":"1051060840310315562135522"},
{"direction":"true","input":"150000000000000000000000","net":"144721330368263208184293","price":"76234063429139383403323324315","tick":"-771","after0":"1100236988854960354203438","after1":"904135631814616873623549"},
{"direction":"false","input":"200000000000000000000000","net":"196331793956899391531371","price":"82454854727345576460955769358","tick":"798","after0":"900915370624605641988848","after1":"1104135631814616873623547"}
]`
