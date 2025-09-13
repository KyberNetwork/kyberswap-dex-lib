package xpress

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

func TestPoolTracker_GetNewPoolState(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip()
	}

	rpcURL := "https://rpc.soniclabs.com"
	multicallAddress := common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11")
	poolAddress := "0xc7723fe3df538f76a063eb5e62867960d236accf"
	helperAddress := "0x38e577290CAf18d07b5719cc9da1E91Bd753f8c0"
	chainId := valueobject.ChainID(146)

	pt := &PoolTracker{
		config: &Config{
			DexId:         DexType,
			HelperAddress: helperAddress,
			ChainId:       chainId,
		},
		ethrpcClient: ethrpc.New(rpcURL).SetMulticallContract(multicallAddress),
	}
	_, err := pt.GetNewPoolState(context.Background(),
		entity.Pool{Address: poolAddress},
		pool.GetNewPoolStateParams{})
	require.NoError(t, err)
}
