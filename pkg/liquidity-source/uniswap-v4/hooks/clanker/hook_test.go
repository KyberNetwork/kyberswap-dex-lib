package clanker

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestHook_Track(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://arbitrum.kyberengineering.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	chainID := valueobject.ChainIDArbitrumOne

	h := &StaticFeeHook{
		rpcClient:     rpcClient,
		hook:          "0xf7ac669593d2d9d01026fa5b756dd5b4f7aaa8cc",
		pool:          "0x3f3ef57297fb9f0a3dca28b15b7b6d8186c0caba8dfc82294d8181da56113a82",
		token0:        common.HexToAddress("0x3079f9fd56c1fbde6f77bcbc387f371513a00b07"),
		clankerCaller: nil,
	}

	h.clankerCaller, h.crankerCallerErr = NewClankerCaller(ClankerAddressByChain[chainID],
		rpcClient.GetETHClient())

	_, err := h.Track(context.Background(), nil)
	require.NoError(t, err)
}
