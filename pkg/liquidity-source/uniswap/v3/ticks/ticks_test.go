package ticks

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsMissingTrieNodeError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"missing trie node", errors.New("missing trie node abcd"), true},
		// This is the case that motivated delegating to the shared classifier: some
		// RPC providers report pruned state with this text instead of "missing trie
		// node", and the fallback-to-latest-block retry in pool_tracker.go's
		// updateState depends on this returning true for it too.
		{"historical state not available", errors.New("historical state 0xabc is not available"), true},
		{"unrelated error", errors.New("connection reset by peer"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsMissingTrieNodeError(tt.err))
		})
	}
}
