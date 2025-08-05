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

	rpcClient := ethrpc.New("https://unichain.drpc.org").
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	p := &entity.Pool{
		Address: "0xeec51c6b1a9e7c4bb4fc4fa9a02fc4fff3fe94efd044f895d98b5bfbd2ff9433",
		Tokens: []*entity.PoolToken{
			{
				Address: "0x078d782b760474a361dda0af3839290b0ef57ad6",
			},
			{
				Address: "0x9151434b16b9763660705744891fa906f660ecc5",
			},
		},
		StaticExtra: "{\"tickSpacing\":1}",
	}

	hookExtra := "{}"

	h := NewHook(&uniswapv4.HookParam{
		Cfg:         &uniswapv4.Config{ChainID: 130},
		RpcClient:   rpcClient,
		HookAddress: common.HexToAddress("0x005af73a245d8171a0550ffae2631f12cc211888"),
		Pool:        p,
	})

	_, err := h.Track(context.Background(), &uniswapv4.HookParam{
		RpcClient: rpcClient,
		Pool:      p,
		HookExtra: hookExtra,
	})
	require.NoError(t, err)
}
