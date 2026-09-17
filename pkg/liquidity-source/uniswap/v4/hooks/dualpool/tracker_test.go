package dualpool

import (
	"context"
	"math/big"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

// Live read of the Twofold NVDA/USDG pool on Robinhood Chain through the hook's
// Track: exercises the PoolKey tuple packing, the distribution tuple[] decoding
// and the StateView slot0 read against the real contracts.
func TestHook_Track_Live(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip()
	}
	rpc := os.Getenv("ROBINHOOD_RPC")
	if rpc == "" {
		rpc = "https://rpc.mainnet.chain.robinhood.com"
	}
	client := ethrpc.New(rpc).SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	p := entity.Pool{
		Address: "0x085e812d2b072f9569c192769fef074eac9b3b519647665c98a7e4119f8fa06b",
		Tokens: []*entity.PoolToken{
			{Address: "0x5fc5360d0400a0fd4f2af552add042d716f1d168", Decimals: 6, Swappable: true},
			{Address: "0xd0601ce157db5bdc3162bbac2a2c8af5320d9eec", Decimals: 18, Swappable: true},
		},
		StaticExtra: `{"0x0":[false,false],"fee":500,"tS":10,"hooks":"0xd1bcbcca41f3bdb6b4812652959c6df725ea2ac0"}`,
	}
	hookAddr := HookAddresses[0]
	param := &uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{StateViewAddress: defaultStateViewRobinhood},
		RpcClient:   client,
		Pool:        &p,
		HookAddress: hookAddr,
		BlockNumber: big.NewInt(55199380),
	}
	h, ok := uniswapv4.GetHook(hookAddr, param)
	require.True(t, ok)

	raw, err := h.Track(context.Background(), param)
	require.NoError(t, err)
	var extra Extra
	require.NoError(t, json.Unmarshal(raw, &extra))
	require.True(t, extra.Live)
	require.Len(t, extra.Buckets, 2)
	require.Equal(t, uint32(500), extra.LpFee)
	require.Equal(t, int32(221902), extra.Tick)
	require.Equal(t, "263339524", extra.Balance0.Dec())
	require.Equal(t, "1140173744005663979", extra.Balance1.Dec())

	reserves, err := h.GetReserves(context.Background(), param)
	require.NoError(t, err)
	require.Equal(t, entity.PoolReserves{"263339524", "1140173744005663979"}, reserves)
}
