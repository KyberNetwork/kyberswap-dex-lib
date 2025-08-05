package bunniv2

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestHook_Track(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://ethereum.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	p := &entity.Pool{
		Address: "0xd9f673912e1da331c9e56c5f0dbc7273c0eb684617939a375ec5e227c62d6707",
		Tokens: []*entity.PoolToken{
			{
				Address: "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48",
			},
			{
				Address: "0xdac17f958d2ee523a2206206994597c13d831ec7",
			},
		},
		StaticExtra: "{\"tickSpacing\":1}",
	}

	cfg := &uniswapv4.Config{
		ChainID: 1,
	}

	hookExtra := "{}"

	h := NewHook(&uniswapv4.HookParam{
		Cfg:         cfg,
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0x000052423c1db6b7ff8641b85a7eefc7b2791888"),
		Pool:        p,
	})

	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		Cfg:       cfg,
		RpcClient: rpcClient,
		Pool:      p,
		HookExtra: hookExtra,
	})
	require.NoError(t, err)
}
