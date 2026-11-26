package shared

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeBasePool is a base pool with tokens only; PoolTokens calls nothing else.
type fakeBasePool struct {
	IBasePool
	tokens []string
}

func (f fakeBasePool) GetTokens() []string { return f.tokens }

func basePools(tokenSets ...[]string) map[string]IBasePool {
	m := make(map[string]IBasePool, len(tokenSets))
	for _, tokens := range tokenSets {
		m[tokens[0]] = fakeBasePool{tokens: tokens}
	}
	return m
}

// PoolTokens lists the pool's own tokens first, in their order (the order of its reserves), then the tokens its
// base pools add, once each and sorted, whatever order the maps iterate in.
func TestPoolTokens(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name      string
		tokens    []string
		basePools map[string]IBasePool
		want      []string
	}{
		{"no base pools", []string{"c", "a", "d", "b"}, nil, []string{"c", "a", "d", "b"}},
		{"empty pool, no base pools", nil, nil, nil},
		{"one base pool", []string{"bpt", "weth"},
			basePools([]string{"bpt", "dai", "usdc", "usdt"}), []string{"bpt", "weth", "dai", "usdc", "usdt"}},
		{"two base pools sharing a token", []string{"bpt", "weth", "bpt2"},
			basePools([]string{"bpt", "dai", "usdc", "usdt"}, []string{"bpt2", "usdc", "frax"}),
			[]string{"bpt", "weth", "bpt2", "dai", "frax", "usdc", "usdt"}},
		{"base pool adds nothing new", []string{"weth", "bpt"},
			basePools([]string{"bpt", "weth"}), []string{"weth", "bpt"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			for range 100 { // the base pools are a map, and the extra tokens come from it
				require.Equal(t, tc.want, PoolTokens(tc.tokens, tc.basePools))
			}
		})
	}
}
