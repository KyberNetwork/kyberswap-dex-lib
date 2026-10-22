package tickerdualpool

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/dualpool"
)

func TestGasEstimateCoversMeasuredFixtureCases(t *testing.T) {
	raw := []byte(capturedGas)
	var fixtures map[string][]int64
	require.NoError(t, json.Unmarshal(raw, &fixtures))
	for name, measured := range fixtures {
		count := 1
		if name == "eightBuckets" {
			count = 8
		}
		h := &Hook{State: fixtureState()}
		h.Pool.Buckets = make([]dualpool.Bucket, count)
		q := Quote{Swap: SwapInfo{Tick: h.Pool.Tick}}
		estimate := h.quoteGas(q) + uniswapv4.DefaultGas.BaseGas
		for _, m := range measured {
			require.GreaterOrEqual(t, estimate, m)
		}
	}
}

// eth_estimateGas through the Kyber router at Monad block 111021450, for two
// buys and two sells per distribution.
const capturedGas = `{
"oneBucket":[1377096,1392376,1376968,1399609],
"eightBuckets":[2428452,2433634,2428318,2439949]
}`
