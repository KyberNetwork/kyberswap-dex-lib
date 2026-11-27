package fewshell_test

import (
	"bytes"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	fewshell "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/few/shell"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
)

// A simulator is stored as msgpack between services: the tracked state and the protocol fee rates read at
// construction must survive it, and the restored hook must quote and update like the original.
func TestHookMsgpackRoundTrip(t *testing.T) {
	hX := `{"pa":"0x147d15e009a63ebed5196ea029679329204f98fd","fw":["0xa250cc729bb3323e7933022a67b52200fe354767",` +
		`"0xef87f4608e601e8564800265aee1c1ffadf73283"],"rs":["16929650201529705921438","29261029832819"],` +
		`"iv":["564000000000000000000","72000000000000"],"bk":["1833000000000000000000","1978000000000000"]}`
	h := fewshell.NewHook(&uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(hX),
		Pool: &entity.Pool{Extra: `{"pF":50331811880}`}}) // the rates come from the pool's pF

	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	defer msgpack.PutEncoder(enc)
	require.NoError(t, enc.Encode(&h))
	dec := msgpack.NewDecoder(&buf)
	defer msgpack.PutDecoder(dec)
	var restored uniswapv4.Hook
	require.NoError(t, dec.Decode(&restored))

	params := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1_000_000_000_000)}
	before, err := h.BeforeSwap(params)
	require.NoError(t, err)
	after, err := restored.BeforeSwap(params)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Equal(t, "-1721", after.DeltaUnspecified.String(), "10 bps survived the round trip")
	restored.UpdateBalance(after.SwapInfo)
	require.NotEqual(t, h.(*fewshell.Hook).Reserves, restored.(*fewshell.Hook).Reserves)
}
