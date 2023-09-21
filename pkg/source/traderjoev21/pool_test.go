package traderjoev21

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
	// https://docs.traderjoexyz.com/deployment-addresses/avalanche#v21
	factoryAddress = "0x8e42f2F4101563bF679975178e880FD87d3eFd4e"
)

func TestGetNewPools(t *testing.T) {
	t.Skip()

	config := &traderjoecommon.Config{
		DexID:          "traderjoev21",
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
	btcbUSDCPool = "0x4224f6F4C9280509724Db2DbAc314621e4465C29"
)

func TestGetPoolState(t *testing.T) {
	t.Skip()

	client := ethrpc.New(rpcURL)
	client.SetMulticallContract(common.HexToAddress(multicallAddress))
	tracker, err := NewPoolTracker(client)
	require.NoError(t, err)

	pool, err := tracker.GetNewPoolState(context.Background(), entity.Pool{Address: btcbUSDCPool}, sourcePool.GetNewPoolStateParams{})
	require.NoError(t, err)

	spew.Dump(pool)
}
