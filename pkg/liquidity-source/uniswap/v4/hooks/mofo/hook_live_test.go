package mofo

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// TestTrack_Live decodes MofoHook.pools(id) for MOFO's ETH pool on Robinhood chain, confirming hookABI's output
// order and types match the deployed contract, and that a tracked pool is never read again (both values are
// written once, in beforeInitialize).
func TestTrack_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://rpc.mainnet.chain.robinhood.com").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	param := &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: HookAddresses[0],
		Pool:        &entity.Pool{Address: "0x3ad7d794ef846fe0831223f373bb07e7b6021618be8bb0aba7b47f9e64a7c6bb"},
	}

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	raw, err := h.Track(context.Background(), param)
	require.NoError(t, err)

	// MOFO's pools were created 2026-09-24T21:07:11Z with a 2% (20000 pip) fee
	assert.Equal(t, Extra{FeePips: 20_000, LaunchedAt: 1790284031}, h.Extra)
	var decoded Extra
	require.NoError(t, json.Unmarshal(raw, &decoded))
	assert.Equal(t, h.Extra, decoded)

	// tracked once: no RPC client needed any more
	again, err := (&Hook{Hook: &uniswapv4.BaseHook{}, Extra: decoded}).Track(context.Background(), &uniswapv4.HookParam{})
	require.NoError(t, err)
	assert.JSONEq(t, string(raw), string(again))

	// a pool id this hook never initialised reads as all zero and is kept unpriceable
	unknown := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err = unknown.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: HookAddresses[0],
		Pool:        &entity.Pool{Address: "0x4a634799be6102208cffca38fa030c3d7c45a22920ee424753b1eacf3b7c1c20"},
	})
	require.NoError(t, err)
	_, err = unknown.BeforeSwap(&uniswapv4.BeforeSwapParams{CalcOut: true})
	assert.ErrorIs(t, err, ErrPoolNotRegistered)
}
