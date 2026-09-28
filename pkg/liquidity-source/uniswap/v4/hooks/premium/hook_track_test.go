package premium

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

// TestTrack_DecodesPoolConfig exercises the RPC decode itself. The other tests inject a
// ready-made HookExtra, so they cannot catch a mis-bound multi-output call: poolConfig
// returns five values and ethrpc decodes those into one struct destination, not five
// pointers. Getting that wrong fails every Track, which silently drops graduated pools.
func TestTrack_DecodesPoolConfig(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("live RPC")
	}

	// $PRM's graduated pool on Robinhood Chain, hook 0x1af6269a… (retired GraduationManager).
	const poolID = "0x093af755db0f7b0570da44c61a5a8915d39f8ab0dd3b7e7ce0771980a9639142"
	hookAddr := common.HexToAddress("0x1af6269a7e53422406ff2410b8ed5590f610aacc")

	h := &Hook{Hook: &uniswapv4.BaseHook{}}
	extra, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient: ethrpc.New("https://rpc.mainnet.chain.robinhood.com").
			SetMulticallContract(common.HexToAddress("0x2cAC2D899eCC914d704FeaAE33ac1bF36277DaD1")),
		HookAddress: hookAddr,
		Pool:        &entity.Pool{Address: poolID},
	})
	require.NoError(t, err)

	var got Hook
	require.NoError(t, json.Unmarshal(extra, &got))
	// The pool is registered and live: memeIsCurrency0 is false (WETH is currency0) and it
	// is not paused. Asserting concrete values, not just "no error", so a decode that
	// silently zeroed every field would still fail here.
	assert.False(t, got.MemeIsCurrency0)
	assert.False(t, got.Paused)
}
