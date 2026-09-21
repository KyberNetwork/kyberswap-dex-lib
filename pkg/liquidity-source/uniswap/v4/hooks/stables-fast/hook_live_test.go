package stablesfast

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// TestTrack_Live reads `feePipsFor` against a real Stables market on Robinhood chain — the
// SDOGE market, whose v4 pool id is below — confirming that the hand-written ABI matches the
// deployed contract rather than merely parsing.
func TestTrack_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://rpc.mainnet.chain.robinhood.com").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862be2a173976CA11"))

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	raw, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: HookAddresses[0],
		Pool: &entity.Pool{
			Address: "0x55db22f2da53a8dd00b9fa6098abd910e65459287a3e43f855ddee595774536e",
		},
	})
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	assert.True(t, h.Tracked)
	assert.True(t, h.Registered, "a live market's pool is registered with the hook")
	// The rate every market ships with. It is owner-mutable, so this asserts the bound the
	// bytecode enforces as well as the shipped value.
	assert.LessOrEqual(t, h.FeePips, maxFeePips.Int64())
	assert.EqualValues(t, 5_000, h.FeePips, "the shipped 0.50% skim")
}

// A pool this hook never registered reads as zero rather than reverting, which is what lets
// a stranger key a PoolKey at this hook and get a hook that does nothing.
func TestTrack_Live_UnregisteredPoolIsFreeNotAnError(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://rpc.mainnet.chain.robinhood.com").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862be2a173976CA11"))

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: HookAddresses[0],
		Pool:        &entity.Pool{Address: common.Hash{}.Hex()},
	})
	require.NoError(t, err)

	assert.True(t, h.Tracked)
	assert.False(t, h.Registered, "the hook is a pure no-op for a pool it never registered")
	assert.EqualValues(t, 0, h.FeePips)
}
