package msgpack

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	fluid "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/fluid/dex-v2"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestFluidV2UpdatedStateRoundTrip(t *testing.T) {
	for fixture, firstOutput := range map[string]int64{"ethereum_d4_dynamic_snapshot.json": 99_971_351, "ethereum_d4_float_snapshot.json": 99_962_579} {
		t.Run(fixture, func(t *testing.T) {
			b, err := os.ReadFile("../liquidity-source/fluid/dex-v2/testdata/" + fixture)
			require.NoError(t, err)
			var f struct {
				Address     string
				Tokens      []string
				Extra       fluid.Extra
				StaticExtra fluid.StaticExtra
			}
			require.NoError(t, json.Unmarshal(b, &f))
			extra, err := json.Marshal(f.Extra)
			require.NoError(t, err)
			static, err := json.Marshal(f.StaticExtra)
			require.NoError(t, err)
			sim, err := fluid.NewPoolSimulator(entity.Pool{Address: f.Address, Type: fluid.DexType, Exchange: fluid.DexType, Extra: string(extra), StaticExtra: string(static), Tokens: []*entity.PoolToken{{Address: f.Tokens[0], Decimals: 6}, {Address: f.Tokens[1], Decimals: 6}}, Reserves: []string{"10000000000", "10000000000"}}, valueobject.ChainIDEthereum)
			require.NoError(t, err)
			params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: f.Tokens[0], Amount: big.NewInt(100_000_000)}, TokenOut: f.Tokens[1]}
			first, err := sim.CalcAmountOut(params)
			require.NoError(t, err)
			require.Equal(t, firstOutput, first.TokenAmountOut.Amount.Int64(), "deployed Tenderly output at fixture block")
			sim.UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn, TokenAmountOut: *first.TokenAmountOut, SwapInfo: first.SwapInfo})
			encoded, err := EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{f.Address: sim})
			require.NoError(t, err)
			decoded, err := DecodePoolSimulatorsMap(encoded)
			require.NoError(t, err)
			restored := decoded[f.Address]
			want, err := sim.CalcAmountOut(params)
			require.NoError(t, err)
			got, err := restored.CalcAmountOut(params)
			require.NoError(t, err)
			require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)
			require.Equal(t, sim.CalculateLimit(), restored.CalculateLimit())
			originalPolicy := any(sim).(pool.ISwapLimitMergePolicy)
			restoredPolicy := restored.(pool.ISwapLimitMergePolicy)
			for key := range sim.CalculateLimit() {
				require.Equal(t, originalPolicy.GetLimitMergePolicy(key), restoredPolicy.GetLimitMergePolicy(key))
			}
		})
	}
}
