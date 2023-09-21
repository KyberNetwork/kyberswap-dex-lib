package traderjoev20

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	sourcePool "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/traderjoecommon"
)

const (
	rpcURL           = "http://localhost:9650/ext/bc/C/rpc"
	multicallAddress = "0xf2fd8219609e28c61a998cc534681f95d2740f61"
	// https://docs.traderjoexyz.com/deployment-addresses/avalanche#v20
	factoryAddress = "0x6E77932A92582f504FF6c4BdbCef7Da6c198aEEf"
	routerAddress  = "0xE3Ffc583dC176575eEA7FD9dF2A7c65F7E23f4C3"
)

func TestGetNewPools(t *testing.T) {
	t.Skip()

	config := &traderjoecommon.Config{
		DexID:          "traderjoev20",
		FactoryAddress: factoryAddress,
		NewPoolLimit:   100,
	}
	client := ethrpc.New(rpcURL)
	client.SetMulticallContract(common.HexToAddress(multicallAddress))
	updater := NewPoolsListUpdater(config, client)

	metadata := traderjoecommon.Metadata{Offset: 0}
	metadataBytes, err := json.Marshal(metadata)
	require.NoError(t, err)

	pools, nextMetadataBytes, err := updater.GetNewPools(context.Background(), metadataBytes)
	require.NoError(t, err)

	nextMetadata := &traderjoecommon.Metadata{}
	require.NoError(t, json.Unmarshal(nextMetadataBytes, nextMetadata))

	fmt.Printf("next offset = %v\n", nextMetadata.Offset)

	require.Truef(t, len(pools) > 0, "there must be pools")

	for _, p := range pools {
		spew.Dump(p)
	}
}

const (
	usdceUSDCPool = "0x18332988456c4bd9aba6698ec748b331516f5a14"
)

func TestGetPoolState(t *testing.T) {
	t.Skip()

	client := ethrpc.New(rpcURL)
	client.SetMulticallContract(common.HexToAddress(multicallAddress))
	tracker, err := NewPoolTracker(client, &traderjoecommon.Config{
		RouterAddress: routerAddress,
	})
	require.NoError(t, err)

	pool, err := tracker.GetNewPoolState(context.Background(), entity.Pool{Address: usdceUSDCPool}, sourcePool.GetNewPoolStateParams{})
	require.NoError(t, err)

	spew.Dump(pool)
}
