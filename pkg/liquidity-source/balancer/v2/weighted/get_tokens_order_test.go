package weighted

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"
)

// GetTokens is shared.PoolTokens (tested in shared/tokens_test.go): the i-th token is the i-th reserve's.
func TestGetTokensPairsWithReserves(t *testing.T) {
	t.Parallel()

	s := newNestedPools(true)
	s.Info.Reserves = make([]*big.Int, len(s.Info.Tokens))
	for i := range s.Info.Reserves {
		s.Info.Reserves[i] = big.NewInt(int64(i + 1))
	}

	tokens := s.GetTokens()
	require.Equal(t, []string{"bpt", "weth", "bpt2", "dai", "frax", "usdc", "usdt"}, tokens)
	for i := range s.GetReserves() {
		require.Equal(t, i, s.GetTokenIndex(tokens[i]), "GetTokens()[%d] is not reserve %d's token", i, i)
	}
}
