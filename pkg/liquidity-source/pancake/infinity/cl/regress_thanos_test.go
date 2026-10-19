package cl

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/pancake/infinity/shared"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// squadswap-cl has no subgraph: ticks come from ModifyLiquidity logs and are read at head. The
// bootstrap must then record head as p.BlockNumber, because FetchPoolTicks re-reads ticks at
// p.BlockNumber and the pool's Initialize block predates every liquidity position (0 ticks).
// Fails if the p.BlockNumber assignment is dropped.
func TestBootstrapLogModeSetsBlockNumber(t *testing.T) {
	rpc := newCLRPC(t)
	rpc.Head = 9000
	ev := shared.CLPoolManagerABI.Events["ModifyLiquidity"]
	data, err := ev.Inputs.NonIndexed().Pack(big.NewInt(-60), big.NewInt(60), big.NewInt(300), [32]byte{})
	require.NoError(t, err)
	rpc.Logs = []ethtypes.Log{{
		Address: common.HexToAddress(clMgr), BlockNumber: 100,
		Topics: []common.Hash{ev.ID, common.HexToHash(clPoolID), common.HexToHash("0x01")}, Data: data,
	}}

	tr := NewPoolTracker(&Config{DexID: "squadswap-cl", CLPoolManagerAddress: clMgr}, rpc.Client(), nil)
	p, err := tr.BootstrapPoolState(t.Context(), entity.Pool{Address: clPoolID, BlockNumber: 50,
		Reserves: entity.PoolReserves{"0", "0"}, StaticExtra: clStaticExtra}, poolpkg.GetNewPoolStateParams{})
	require.NoError(t, err)
	require.EqualValues(t, 9000, p.BlockNumber)

	p, err = tr.FetchPoolTicks(t.Context(), p)
	require.NoError(t, err)
	require.Contains(t, p.Extra, `"index":-60`)
	require.Contains(t, p.Extra, `"index":60`)
	require.Equal(t, "0x2328", rpc.Blocks()[len(rpc.Blocks())-1]) // refetch pinned to head (9000)
}
