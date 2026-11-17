package v2

import (
	"math/big"
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
)

// TestPinnedReadStaysAtThePin: a read pinned to a block is dated by it, and its later rounds are
// read there, though the first round's multicall reports the block after, as Etherlink's does.
func TestPinnedReadStaysAtThePin(t *testing.T) {
	tr, err := NewPoolTracker(&shared.Config{}, ethrpc.New("http://127.0.0.1:1"))
	require.NoError(t, err)

	later := map[int]*big.Int{}
	exec := func(req *ethrpc.Request, round int) (*big.Int, error) {
		if round == 0 {
			return new(big.Int).Add(req.BlockNumber, big.NewInt(1)), nil
		}
		later[round] = req.BlockNumber
		return nil, nil
	}
	vaults := []shared.VaultInfo{
		{VaultAddress: "0x1", AssetAddress: "0xa", QuoteAmount: big.NewInt(1)},
		{VaultAddress: "0x2", AssetAddress: "0xb", QuoteAmount: big.NewInt(1)},
	}
	_, bn, err := tr.getPoolData(t.Context(), "0xp", "0xe", "0xc", vaults, nil, big.NewInt(100), exec)
	require.NoError(t, err)
	assert.EqualValues(t, 100, bn)
	require.Contains(t, later, 3)
	assert.EqualValues(t, 100, later[3].Uint64())
}
