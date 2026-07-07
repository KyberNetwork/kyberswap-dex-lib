package v1

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

// TestPoolFactoryDecodePoolCreated decodes a real PoolDeployed log
// (tx 0xf8bd06384ffcb61d8946778f4047b6d744a404cc1d9109fec178a1e27c4698e2, Unichain) and checks the result matches
// what PoolsListUpdater.initPools produces for the same pool via plain RPC,
// confirming the event-driven decode path stays in sync with the RPC-driven
// backup path.
func TestPoolFactoryDecodePoolCreated(t *testing.T) {
	t.Parallel()
	if os.Getenv("CI") != "" {
		t.Skip("Skipping testing in CI environment")
	}

	ethrpcClient := ethrpc.New("https://unichain.drpc.org").
		SetMulticallContract(common.HexToAddress("0xcA11bde05977b3631167028862bE2a173976CA11"))

	config := &shared.Config{
		DexID:          DexType,
		FactoryAddress: "0x45b146bc07c9985589b52df651310e75c6be066a",
	}

	log := types.Log{
		Address: common.HexToAddress("0x45b146bc07c9985589b52df651310e75c6be066a"),
		Topics: []common.Hash{
			common.HexToHash("0x5f7560a5797edc6f72421362defa094d690eb9f7ced3cc5a5c13383502e4fcc5"),
			common.HexToHash("0x000000000000000000000000078d782b760474a361dda0af3839290b0ef57ad6"),
			common.HexToHash("0x0000000000000000000000004200000000000000000000000000000000000006"),
			common.HexToHash("0x0000000000000000000000003eb6f25a0a879e5a11270d3bc3c17efb6d41b519"),
		},
		Data:        common.FromHex("0x000000000000000000000000cfe91b0b6412e1c8197650c7f28d741fbaaaa8a8"),
		BlockNumber: 18914816,
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
