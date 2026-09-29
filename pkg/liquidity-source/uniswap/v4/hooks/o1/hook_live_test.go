package o1

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

// TestTrack_Live decodes poolConfig against a real o1 launch on Base (poolId found
// via a PoolRegistered log at block 50719550), confirming LaunchHookABI actually
// matches the deployed contract's poolConfig layout.
func TestTrack_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://mainnet.base.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))
	hookAddr := common.HexToAddress("0x1f91c998e7c2f4b690d75bdbf6502bdcd6e02acc")

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: hookAddr,
		Pool: &entity.Pool{
			Address: "0xd980e58458d4fc3055ce4220b0015c186a3f409a8404f3fbdb6096e5b492da0e",
		},
	})
	require.NoError(t, err)

	// Values cross-checked by hand against `cast call ... poolConfig(bytes32)` on 2026-09-06.
	assert.False(t, h.TokenIsCurrency0)
	assert.Equal(t, int64(100), h.BaseFeeBps)
	assert.Equal(t, int64(9900), h.AntiSnipeStartTotalBps)
	assert.Equal(t, int64(20), h.AntiSnipeWindowSeconds)
	assert.Equal(t, int64(1788228447), h.LaunchTime)
}
