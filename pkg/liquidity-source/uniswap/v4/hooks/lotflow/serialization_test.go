package lotflow_test

import (
	"math/big"
	"testing"

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
		encoded, err := msgpack.EncodePoolSimulatorsMap(map[string]pool.IPoolSimulator{ep.Address: sim})
		require.NoError(t, err)
		decoded, err := msgpack.DecodePoolSimulatorsMap(encoded)
		require.NoError(t, err)
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{
			Token: ep.Tokens[0].Address, Amount: big.NewInt(100_000_000)}, TokenOut: ep.Tokens[1].Address}
		want, err := sim.CalcAmountOut(params)
		require.NoError(t, err)
		got, err := decoded[ep.Address].CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, want.TokenAmountOut, got.TokenAmountOut)
		require.Equal(t, sim.GetMetaInfo(params.TokenAmountIn.Token, params.TokenOut),
			decoded[ep.Address].GetMetaInfo(params.TokenAmountIn.Token, params.TokenOut))
		decoded[ep.Address].UpdateBalance(pool.UpdateBalanceParams{TokenAmountIn: params.TokenAmountIn,
			TokenAmountOut: *got.TokenAmountOut, SwapInfo: got.SwapInfo})
		_, err = decoded[ep.Address].CalcAmountOut(params)
		require.NoError(t, err, "decoded simulator can quote the next swap")
		unchanged, err := sim.CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, want.TokenAmountOut, unchanged.TokenAmountOut)
	}
}
