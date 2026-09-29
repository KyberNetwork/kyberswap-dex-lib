package ponsv2

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

// TestTrack_Live decodes `launches` against a real graduated pons-v2 pool on Robinhood
// chain (found via a live PoolRegistered log from the hook contract itself), confirming
// hookABI's field order/types actually match the deployed contract -- not just that the
// hand-written ABI JSON compiles.
func TestTrack_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://rpc.mainnet.chain.robinhood.com").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862be2a173976CA11"))
	hookAddr := common.HexToAddress("0xE5e702641Ea86F4ae6cC3cDaeD2B886f976Be044")

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient:   rpcClient,
		HookAddress: hookAddr,
		Pool: &entity.Pool{
			Address: "0xad66e295ab58f146c1c19901784444be40ebfa64d49f3311168ba2f27a75f56c",
		},
	})
	require.NoError(t, err)

	// Values cross-checked by hand against a raw eth_call to launches(bytes32) on
	// 2026-09-06: hookFeeBps=100, creatorTaxBps=0 match the hook's constructor defaults
	// (hookFeeBps=100), and this launch chose no creator tax.
	assert.True(t, h.Registered)
	assert.Equal(t, int64(100), h.FeeBps)
}
