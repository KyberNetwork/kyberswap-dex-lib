package prmfun_test

import (
	"bytes"
	"encoding/json"
	"math/big"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	prmfun "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/prm-fun"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// Downstream quote services persist simulators, including the native/ERC-20 flag.
func TestAllPairsMsgpackRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("testdata/fork-quotes.json")
	require.NoError(t, err)
	var fixtures []struct {
		Pair   string
		Pool   entity.Pool
		Quotes []struct {
			Buy   bool
			Input string
		}
	}
	require.NoError(t, json.Unmarshal(raw, &fixtures))
	for _, f := range fixtures {
		t.Run(f.Pair, func(t *testing.T) {
			s, err := prmfun.NewPoolSimulator(f.Pool)
			require.NoError(t, err)
			var original pool.IPoolSimulator = s
			var buf bytes.Buffer
			enc := msgpack.NewEncoder(&buf)
			defer msgpack.PutEncoder(enc)
			require.NoError(t, enc.Encode(&original))
			dec := msgpack.NewDecoder(&buf)
			defer msgpack.PutDecoder(dec)
			var restored pool.IPoolSimulator
			require.NoError(t, dec.Decode(&restored))
			for _, q := range f.Quotes {
				input, output := f.Pool.Tokens[1].Address, f.Pool.Tokens[0].Address
				if q.Buy {
					input, output = output, input
				}
				amount, ok := new(big.Int).SetString(q.Input, 10)
				require.True(t, ok)
				params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: input, Amount: amount}, TokenOut: output}
				before, err := original.CalcAmountOut(params)
				require.NoError(t, err)
				after, err := restored.CalcAmountOut(params)
				require.NoError(t, err)
				require.Equal(t, before, after)
				require.Equal(t, original.GetMetaInfo(input, output), restored.GetMetaInfo(input, output))
				restored.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: after.SwapInfo})
				original.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: before.SwapInfo})
				require.Equal(t, original.GetReserves(), restored.GetReserves())
			}
		})
	}
}
