package eth_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/eth"
)

func TestIsPrunedStateError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"unrelated error", errors.New("connection reset by peer"), false},
		{"trie node", errors.New("missing trie node abcd"), true},
		{"historical state", errors.New("historical state 0xabc is not available"), true},
		{"archive mode", errors.New("your node is not running in archive mode"), true},
		{"archive node", errors.New("this data is only available on an archive node"), true},
		{"header not found is not treated as pruned", errors.New("header not found"), false},
		{"state not available", errors.New("state not available at this block"), true},
		{"state unavailable", errors.New("state unavailable, try a different block"), true},
		{"database is pruned", errors.New("the requested data is not available because the database is pruned"), true},
		{"storage trie", errors.New("could not open storage trie"), true},
		{"state at block is pruned", errors.New("state at block #49613289 is pruned"), true},
		{"case insensitive", errors.New("MISSING TRIE NODE"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, eth.IsPrunedStateError(tt.err))
		})
	}
}
