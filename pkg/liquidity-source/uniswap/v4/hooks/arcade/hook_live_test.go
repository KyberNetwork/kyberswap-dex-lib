package arcade

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

// TestTrack_Live decodes the hook getters against a real CLANKER launch on Arc mainnet,
// confirming the ABI field order and types match the deployed contract.
// Values cross-checked against direct eth_calls on 2026-09-15.
func TestTrack_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://rpc.arc-scan.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: HookAddresses[0],
		Pool: &entity.Pool{
			Address: "0xe9ab261f1c77caeefd37aec54859484c93c8856b95739e779e4827616cab54fe",
			Tokens: []*entity.PoolToken{
				{Address: "0x3600000000000000000000000000000000000000"}, // USDC
				{Address: "0x43caace3d7bc72b25e32d2b81b1ee28f447e4ffd"}, // launch token
			},
		},
	})
	require.NoError(t, err)

	assert.True(t, h.Tracked)
	assert.Equal(t, uint8(ModeClanker), h.Mode)
	assert.Equal(t, uint8(StatusGraduated), h.Status)
	assert.True(t, h.UsdcIsCurrency0)
	assert.True(t, h.QuoteIsCurrency0)
	assert.False(t, h.ObsInit)
	assert.Equal(t, int64(1789446515), h.LaunchedAt)
	assert.True(t, h.BuyCapEnabled)
}
