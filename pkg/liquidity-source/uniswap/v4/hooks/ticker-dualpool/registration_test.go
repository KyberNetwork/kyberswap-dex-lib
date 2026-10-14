package tickerdualpool

import (
	"context"
	"math/big"
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestProductionHookFactoryLookup(t *testing.T) {
	raw, err := json.Marshal(Snapshot{State: fixtureState(), BlockNumber: 123})
	require.NoError(t, err)
	hook, ok := uniswapv4.GetHook(productionHookAddress, &uniswapv4.HookParam{
		Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDMonad}, HookExtra: raw,
	})
	require.True(t, ok)
	h, ok := hook.(*Hook)
	require.True(t, ok)
	require.Equal(t, productionHookAddress, h.address)
	require.Equal(t, valueobject.ExchangeUniswapV4TickerDualPool, h.GetExchange())
	require.True(t, h.AllowEmptyTicks())
	result, err := h.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1001)})
	require.NoError(t, err)
	require.Equal(t, "984", result.SwapInfo.(Quote).NetOutput.Dec())

	malformed, ok := uniswapv4.GetHook(productionHookAddress, &uniswapv4.HookParam{
		Cfg: &uniswapv4.Config{ChainID: valueobject.ChainIDMonad}, HookExtra: []byte("{"),
	})
	require.True(t, ok)
	_, err = malformed.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1001)})
	require.Error(t, err)
}

func TestProductionHookRejectsMissingOrWrongChain(t *testing.T) {
	raw, err := json.Marshal(Snapshot{State: fixtureState(), BlockNumber: 123})
	require.NoError(t, err)
	for _, cfg := range []*uniswapv4.Config{nil, {}, {ChainID: valueobject.ChainIDEthereum}} {
		params := &uniswapv4.HookParam{Cfg: cfg, HookExtra: raw}
		hook, ok := uniswapv4.GetHook(productionHookAddress, params)
		require.True(t, ok)
		_, err := hook.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true, ZeroForOne: true, AmountSpecified: big.NewInt(1001)})
		require.ErrorIs(t, err, ErrInvalidState)
		_, err = hook.Track(context.Background(), params)
		require.ErrorIs(t, err, ErrInvalidState, "reject unsupported chain before any RPC")
	}
}
