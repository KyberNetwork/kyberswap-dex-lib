package v2

import (
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/euler-swap/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/testutil"
)

// maxInitCode is EIP-3860's limit, which geth's state transition applies to a to-less eth_call
// (core/vm.CheckMaxInitCodeSize) before running it: a plain error, no revert, no data.
const maxInitCode = 49152

// initCodeNode answers a lens call like a Shanghai node: an oversize one fails outright, a smaller
// one reads every pool with empty rounds.
func initCodeNode(t *testing.T, lensCalls, oversize *atomic.Int64) *testutil.NodeStub {
	return testutil.NewNodeStub(t, 1, 100, func(c testutil.NodeCall) ([]byte, error) {
		if c.To != nil {
			return nil, errors.New("no multicall here")
		}
		lensCalls.Add(1)
		if len(c.Data) > maxInitCode {
			oversize.Add(1)
			return nil, fmt.Errorf("max initcode size exceeded: code size %d limit %d", len(c.Data), maxInitCode)
		}
		out, err := lens.PackResult(big.NewInt(100), make([]poolReads, len(lensPools(t, c.Data))))
		require.NoError(t, err)
		return nil, testutil.Revert(out)
	})
}

func manyPools(n int) []entity.Pool {
	pools := make([]entity.Pool, n)
	for i := range pools {
		pools[i] = entity.Pool{
			Address: common.BigToAddress(big.NewInt(int64(0x1000 + i))).Hex(),
			Tokens:  []*entity.PoolToken{{Address: "0x00000000000000000000000000000000000000aa", Decimals: 6}, {Address: "0x00000000000000000000000000000000000000bb", Decimals: 18}},
			StaticExtra: fmt.Sprintf(`{"sv0":"%s","sv1":"%s","ea":"%s","evc":"0x0000000000000000000000000000000000000c0c"}`,
				common.BigToAddress(big.NewInt(int64(0x2000+i))).Hex(), common.BigToAddress(big.NewInt(int64(0x3000+i))).Hex(),
				common.BigToAddress(big.NewInt(int64(0x4000+i))).Hex()),
		}
	}
	return pools
}

// TestLensBatchOverInitCodeLimitIsSplit: GetNewPoolStates hands readLens every pool of a batch in
// one call. Past ~83 pools of two vaults the call data passes EIP-3860's 49152 bytes, which a node
// refuses before running it; the batch is then read in smaller parts, not given up for multicalls.
func TestLensBatchOverInitCodeLimitIsSplit(t *testing.T) {
	var lensCalls, oversize atomic.Int64
	tr, err := NewPoolTracker(&shared.Config{UseLens: true}, initCodeNode(t, &lensCalls, &oversize).Client())
	require.NoError(t, err)

	pools := manyPools(100)
	var inputs []poolInput
	for _, p := range pools {
		se, vaults, err := poolVaults(p)
		require.NoError(t, err)
		inputs = append(inputs, lensInput(p.Address, se, vaults))
	}
	full, err := lens.PackCall(inputs)
	require.NoError(t, err)
	half, err := lens.PackCall(inputs[:50])
	require.NoError(t, err)
	require.Greater(t, len(full), maxInitCode)
	require.Less(t, len(half), maxInitCode)

	_, reads, err := tr.readLens(t.Context(), inputs, nil, 0)
	assert.NoError(t, err, "a batch over the init code limit should be read in smaller parts, not given up")
	assert.Len(t, reads, len(inputs))
}
