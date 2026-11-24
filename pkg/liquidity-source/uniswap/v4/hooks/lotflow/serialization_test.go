package lotflow_test

import (
	"bytes"
	"testing"

	"github.com/holiman/uint256"
	"github.com/klauspost/compress/snappy"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/lotflow"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// Exercise the real registry and interface serialization used between Kyber services, rather than
// serializing only a concrete hook. Both exact quoting and executor metadata must survive it.
func TestPoolSimulatorSerialization(t *testing.T) {
	for _, mq := range []bool{false, true} {
		ep := lotflow.FixturePoolForTest(t, mq)
		sim, err := uniswapv4.NewPoolSimulator(ep, valueobject.ChainIDRobinhood)
		require.NoError(t, err)
		var buf bytes.Buffer
		zw := snappy.NewBufferedWriter(&buf)
		en := msgpack.NewEncoder(zw)
		require.NoError(t, en.Encode(map[string]pool.IPoolSim{ep.Address: sim}))
		msgpack.PutEncoder(en)
		require.NoError(t, zw.Close())
		decoded := map[string]pool.IPoolSim{}
		de := msgpack.NewDecoder(snappy.NewReader(bytes.NewReader(buf.Bytes())))
		require.NoError(t, de.Decode(&decoded))
		msgpack.PutDecoder(de)
		var amountIn uint256.Int
		amountIn.SetUint64(100_000_000)
		params := pool.CalcAmtOutParams{TokenAmountIn: pool.TokenAmt{Token: ep.Tokens[0].Address, Amount: amountIn}, TokenOut: ep.Tokens[1].Address}
		want, err := sim.CalcAmtOut(params)
		require.NoError(t, err)
		got, err := decoded[ep.Address].CalcAmtOut(params)
		require.NoError(t, err)
		require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)
		require.Equal(t, sim.GetMetaInfo(params.TokenAmountIn.Token, params.TokenOut),
			decoded[ep.Address].GetMetaInfo(params.TokenAmountIn.Token, params.TokenOut))
		decoded[ep.Address].UpdateState(pool.UpdateStateParams{TokenAmountIn: params.TokenAmountIn,
			TokenAmountOut: *got.TokenAmountOut, SwapInfo: got.SwapInfo})
		_, err = decoded[ep.Address].CalcAmtOut(params)
		require.NoError(t, err, "decoded simulator can quote the next swap")
		unchanged, err := sim.CalcAmtOut(params)
		require.NoError(t, err)
		require.Equal(t, want.TokenAmountOut, unchanged.TokenAmountOut)
	}
}
