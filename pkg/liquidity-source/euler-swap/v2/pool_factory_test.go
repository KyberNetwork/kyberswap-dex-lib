package v2

import (
	"context"
	"os"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
)

// TestPoolFactoryDecodePoolCreated decodes a real PoolRegistered log
// (tx 0x99afdf7e73c3eea928c8a6b816dc4e91c74a42813dcca25a652622687d4ec058, mainnet) and checks the result matches
// what PoolsListUpdater.initPools produces for the same pool via plain RPC,
// confirming the event-driven decode path stays in sync with the RPC-driven
// backup path.
func TestPoolFactoryDecodePoolCreated(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	ethrpcClient := ethrpc.New("https://rpc.mevblocker.io").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	config := &shared.Config{
		DexID:          DexType,
		FactoryAddress: "0x5fccb84363f020c0cade052c9c654aabf932814a",
	}

	log := types.Log{
		Address: common.HexToAddress("0x5fccb84363f020c0cade052c9c654aabf932814a"),
		Topics: []common.Hash{
			common.HexToHash("0x119fbab0d994e1e8e8de13032b258cc20a84d8da97757668fe40686cebb47ac4"),
			common.HexToHash("0x00000000000000000000000008efcc2f3e61185d0ea7f8830b3fec9bfa2ee313"),
			common.HexToHash("0x000000000000000000000000a0b86991c6218b36c1d19d4a2e9eb0ce3606eb48"),
			common.HexToHash("0x000000000000000000000000006d9f269695ad9eb8f727f042ee380684332917"),
		},
		Data:        common.FromHex("0x0000000000000000000000004ddccfda0cdf6e60866e5fe3e11995fe4ad0e8a8000000000000000000000000c073abfcaa318157340bda9afceaf749f9c1a43e000000000000000000000000797dd80692c3b2dadabce8e30c07fde5307d48a9000000000000000000000000c073abfcaa318157340bda9afceaf749f9c1a43e000000000000000000000000797dd80692c3b2dadabce8e30c07fde5307d48a9000000000000000000000000006d9f269695ad9eb8f727f042ee38068433291700000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000"),
		BlockNumber: 25450344,
	}

	factory := NewPoolFactory(config, ethrpcClient)
	require.True(t, factory.IsEventSupported(log.Topics[0]))

	pool, err := factory.DecodePoolCreated(log)
	require.NoError(t, err)
	require.NotNil(t, pool)

	listUpdater := NewPoolsListUpdater(config, ethrpcClient)
	rpcPools, err := listUpdater.initPools(context.Background(), []common.Address{
		common.HexToAddress(pool.Address),
	})
	require.NoError(t, err)
	require.Len(t, rpcPools, 1)

	require.Equal(t, rpcPools[0].Address, pool.Address)
	require.Equal(t, rpcPools[0].StaticExtra, pool.StaticExtra)
	require.Equal(t, rpcPools[0].Tokens[0].Address, pool.Tokens[0].Address)
	require.Equal(t, rpcPools[0].Tokens[1].Address, pool.Tokens[1].Address)
}
