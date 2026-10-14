package tickerdualpool

import (
	"encoding/json"
	"strconv"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
)

func TestDirectionalProtocolFeesMatchMonadFork(t *testing.T) {
	raw := []byte(capturedProtocolFees)
	var rows []map[string]string
	require.NoError(t, json.Unmarshal(raw, &rows))
	require.Len(t, rows, 4)
	for _, f := range rows {
		t.Run(f["direction"]+"/"+f["input"], func(t *testing.T) {
			s := fixtureState()
			s.Pool.Balance0 = uint256.MustFromDecimal(f["before0"])
			s.Total0 = s.Pool.Balance0.Clone()
			s.Pool.Balance1 = uint256.MustFromDecimal(f["before1"])
			s.Total1 = s.Pool.Balance1.Clone()
			s.Pool.SqrtPriceX96 = uint256.MustFromDecimal(f["initialPrice"])
			tick, err := strconv.Atoi(f["initialTick"])
			require.NoError(t, err)
			s.Pool.Tick = int32(tick)
			protocol, err := strconv.ParseUint(f["protocol"], 10, 32)
			require.NoError(t, err)
			s.Pool.ProtocolFee = uint32(protocol)
			s.Pool.Buckets = []dualpool.Bucket{{TickLower: 312380, TickUpper: 332380, WeightBps: 10000}}
			q, err := QuoteExactInput(s, f["direction"] == "true", uint256.MustFromDecimal(f["input"]))
			require.NoError(t, err)
			require.Equal(t, f["net"], q.NetOutput.Dec())
			require.Equal(t, f["fee"], q.HookFee.Dec())
			require.Equal(t, f["tax"], q.CreatorTax.Dec())
			require.Equal(t, f["price"], q.Swap.SqrtPriceX96.Dec())
			require.Equal(t, f["tick"], strconv.Itoa(int(q.Swap.Tick)))
		})
	}
}

// Captured contract results; preserved from protocol-fees.json.
const capturedProtocolFees = `[
{"direction":"true","input":"100000000","before0":"9999999999","before1":"1000000000000000000000000","initialPrice":"792359631301295627757723824548136721","initialTick":"322380","protocol":"4096250","net":"9791327569279233519990","fee":"99404340804865314923","tax":"49702170402432657461","price":"789260629333123021886236102138641953","tick":"322301"},
{"direction":"true","input":"1000000000","before0":"9999999999","before1":"1000000000000000000000000","initialPrice":"792359631301295627757723824548136721","initialTick":"322380","protocol":"4096250","net":"94583926413983930413687","fee":"960242907756182034656","tax":"480121453878091017328","price":"762423366577184658169390427626484353","tick":"321609"},
{"direction":"false","input":"1000000000000000000000","before0":"9999999999","before1":"1000000000000000000000000","initialPrice":"792359631301295627757723824548136721","initialTick":"322380","protocol":"4096250","net":"9814687","fee":"99641","tax":"49820","price":"792670453860477147870525853005265434","tick":"322387"},
{"direction":"false","input":"100000000000000000000000","before0":"9999999999","before1":"1000000000000000000000000","initialPrice":"792359631301295627757723824548136721","initialTick":"322380","protocol":"4096250","net":"944791835","fee":"9591795","tax":"4795897","price":"823441887219447639037926344581738215","tick":"323149"}
]`
