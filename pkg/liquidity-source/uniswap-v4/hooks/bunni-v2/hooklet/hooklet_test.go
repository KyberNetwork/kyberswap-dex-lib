package hooklet

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

func TestHooklet_Track(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	rpcClient := ethrpc.New("https://unichain.drpc.org").
		SetMulticallContract(common.HexToAddress("0xca11bde05977b3631167028862be2a173976ca11"))

	h := feeOverrideHooklet{}

	res, err := h.Track(context.Background(), HookletParams{
		RpcClient:      rpcClient,
		HookletAddress: common.HexToAddress("0x0000e819b8A536Cf8e5d70B9C49256911033000C"),
		PoolId:         common.HexToHash("0xe7b110a6045c9e17b97902a414604b96ef0ccd227abbb0f0761da09437522e4d"),
		HookletExtra:   "",
	})

	require.NotNil(t, res)
	require.NoError(t, err)
}
