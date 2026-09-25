package inverse_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/goccy/go-json"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/hooks/inverse"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/msgpack"
)

func TestHookMsgpackRoundTrip(t *testing.T) {
	raw, err := os.ReadFile("testdata/solidity.json")
	require.NoError(t, err)
	var vs []struct {
		Before json.RawMessage `json:"before"`
		Zero   bool            `json:"zeroForOne"`
		Input  uint256.Int     `json:"amountIn"`
	}
	require.NoError(t, json.Unmarshal(raw, &vs))
	h := inverse.NewHook(&uniswapv4.HookParam{HookExtra: uniswapv4.HookExtra(vs[0].Before)})
	var buf bytes.Buffer
	enc := msgpack.NewEncoder(&buf)
	defer msgpack.PutEncoder(enc)
	require.NoError(t, enc.Encode(&h))
	dec := msgpack.NewDecoder(&buf)
	defer msgpack.PutDecoder(dec)
	var restored uniswapv4.Hook
	require.NoError(t, dec.Decode(&restored))
	p := &uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: vs[0].Zero, AmountSpecified: vs[0].Input.ToBig()}
	before, err := h.BeforeSwap(p)
	require.NoError(t, err)
	after, err := restored.BeforeSwap(p)
	require.NoError(t, err)
	require.Equal(t, before, after)
	restored.UpdateBalance(after.SwapInfo)
	require.NotEqual(t, h.(*inverse.Hook).State, restored.(*inverse.Hook).State)
}
