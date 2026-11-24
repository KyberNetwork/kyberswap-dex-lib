package evplusai_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/klauspost/compress/snappy"
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
	decoded := map[string]pool.IPoolSim{f.Pool.Address: sim}
	// Tagged interface values decode to pointers; saving a restored simulator
	// must retain the wrapper type on a second roundtrip as well.
	for range 2 {
		var buf bytes.Buffer
		zw := snappy.NewBufferedWriter(&buf)
		en := msgpack.NewEncoder(zw)
		require.NoError(t, en.Encode(decoded))
		msgpack.PutEncoder(en)
		require.NoError(t, zw.Close())
		decoded = map[string]pool.IPoolSim{}
		de := msgpack.NewDecoder(snappy.NewReader(bytes.NewReader(buf.Bytes())))
		require.NoError(t, de.Decode(&decoded))
		msgpack.PutDecoder(de)
	}
	restored := decoded[f.Pool.Address].(*uniswapv4.PoolSimulator)
	var amtOut uint256.Int
	amtOut.SetUint64(100000000000000000)
	result, err := restored.CalcAmtIn(pool.CalcAmtInParams{TokenAmountOut: pool.TokenAmt{Token: f.Pool.Tokens[0].Address, Amount: amtOut}, TokenIn: f.Pool.Tokens[1].Address})
	require.NoError(t, err)
	require.Equal(t, q.Expected, result.TokenAmountIn.Amount.String())
}
