package thogprop

import (
	"bytes"
	"testing"

	"github.com/KyberNetwork/msgpack/v5"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

func TestSimulatorMsgpackPreservesSharesAndUpdatedInventory(t *testing.T) {
	s := stateFrom(t, fixture(t, "execution-model3.json.gz").Snapshots[1])
	sim, err := NewPoolSimulator(entityFrom(t, s))
	require.NoError(t, err)
	// Exercise non-unit shares, including after UpdateBalance. Some directions
	// legitimately exceed the read-only capacity after the multiplier changes.
	checked := 0
	for _, c := range fixture(t, "execution-model3.json.gz").Snapshots[1].Cases {
		if !c.Success || c.Method != "execution" || (c.InputIndex < 8 && c.OutputIndex < 8) {
			continue
		}
		amount := number(t, c.Amount)
		params := pool.CalcAmountOutParams{TokenAmountIn: pool.TokenAmount{Token: s.Tokens[c.InputIndex].Address, Amount: amount.ToBig()}, TokenOut: s.Tokens[c.OutputIndex].Address}
		expected, err := sim.CalcAmountOut(params)
		if err != nil {
			continue
		}
		var buffer bytes.Buffer
		encoder := msgpack.NewEncoder(&buffer)
		encoder.IncludeUnexported(true)
		encoder.SetForceAsArray(true)
		require.NoError(t, encoder.Encode(sim))
		var decoded PoolSimulator
		decoder := msgpack.NewDecoder(&buffer)
		decoder.IncludeUnexported(true)
		decoder.SetForceAsArray(true)
		require.NoError(t, decoder.Decode(&decoded))
		require.Equal(t, sim.state, decoded.state)
		actual, err := decoded.CalcAmountOut(params)
		require.NoError(t, err)
		require.Equal(t, expected, actual)
		decoded.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: actual.SwapInfo})
		sim.UpdateBalance(pool.UpdateBalanceParams{SwapInfo: expected.SwapInfo})
		require.Equal(t, sim.state, decoded.state)
		checked++
		if checked == 3 {
			break
		}
	}
	require.Equal(t, 3, checked)
}
