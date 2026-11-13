package stable

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// GetTokens must list the pool's own tokens first, in Info.Tokens order, so that GetTokens()[i] pairs
// with GetReserves()[i] as on other sources. The base pools' tokens follow, once each and sorted, so
// the order does not depend on map iteration.
func TestGetTokensOrderIsDeterministicAndParallelToReserves(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name               string
		withSecondBasePool bool
		want               []string
	}{
		{"one base pool", false, []string{"bpt", "weth", "dai", "usdc", "usdt"}},
		{"two base pools", true, []string{"bpt", "weth", "bpt2", "dai", "frax", "usdc", "usdt"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := newNestedPools(tc.withSecondBasePool)
			s.Info.Reserves = make([]*big.Int, len(s.Info.Tokens))
			for i := range s.Info.Reserves {
				s.Info.Reserves[i] = big.NewInt(int64(i + 1))
			}

			for range 100 { // the base pools and token sets are maps; the order must not follow them
				tokens := s.GetTokens()
				require.Equal(t, tc.want, tokens)

				reserves := s.GetReserves()
				require.Len(t, reserves, len(s.Info.Tokens))
				for i := range reserves {
					assert.Equal(t, i, s.GetTokenIndex(tokens[i]), "GetTokens()[%d] is not reserve %d's token", i, i)
				}
			}
		})
	}
}

func TestGetTokensWithoutBasePoolsIsInfoTokens(t *testing.T) {
	t.Parallel()

	s := newNestedPools(false)
	s.basePools = nil
	s.Info.Tokens = []string{"c", "a", "d", "b"}

	for range 100 {
		require.Equal(t, []string{"c", "a", "d", "b"}, s.GetTokens())
	}
}
