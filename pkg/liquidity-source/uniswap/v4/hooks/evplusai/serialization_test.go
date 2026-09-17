package evplusai_test

import (
	"math/big"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestSimulatorSerialization(t *testing.T) {
	data, err := os.ReadFile("testdata/quotes.json")
	require.NoError(t, err)
	var f struct {
		Pool   entity.Pool `json:"pool"`
		Quotes []struct {
			Extra    json.RawMessage `json:"hookExtra"`
			Name     string          `json:"name"`
			Expected string          `json:"expected"`
		} `json:"quotes"`
	}
	require.NoError(t, json.Unmarshal(data, &f))
	q := f.Quotes[15] // exact-output word-boundary regression, max buy budget
	require.Equal(t, "min-max/sell=false/exactIn=false/large=true", q.Name)
	var ext uniswapv4.Extra
	require.NoError(t, json.Unmarshal([]byte(f.Pool.Extra), &ext))
	ext.HookExtra = q.Extra
	raw, err := json.Marshal(ext)
	require.NoError(t, err)
	f.Pool.Extra = string(raw)
	sim, err := uniswapv4.NewPoolSimulator(f.Pool, valueobject.ChainIDRobinhood)
	require.NoError(t, err)
	encoded, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{f.Pool.Address: sim})
	require.NoError(t, err)
	decoded, err := msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	// Tagged interface values decode to pointers; saving a restored simulator
	// must retain the wrapper type on a second roundtrip as well.
	encoded, err = msgpack.EncodePoolSimulatorsMap(decoded)
	require.NoError(t, err)
	decoded, err = msgpack.DecodePoolSimulatorsMap(encoded)
	require.NoError(t, err)
	restored := decoded[f.Pool.Address].(*uniswapv4.PoolSimulator)
	require.False(t, restored.V3Pool.ExactTickTraversal)
	result, err := restored.CalcAmountIn(pool.CalcAmountInParams{TokenAmountOut: pool.TokenAmount{Token: f.Pool.Tokens[0].Address, Amount: big.NewInt(100000000000000000)}, TokenIn: f.Pool.Tokens[1].Address})
	require.NoError(t, err)
	require.Equal(t, q.Expected, result.TokenAmountIn.Amount.String())
}
