package eth

import "strings"

// prunedStateErrSubstrings matches RPC error text indicating a node no longer serves
// state (or history) for the requested block - because it pruned it, isn't running in
// archive mode, or otherwise doesn't retain it. Substrings, not exact strings, since
// RPC providers phrase this differently ("missing trie node" vs "historical state ...
// is not available" have both been observed from different providers for the same
// underlying condition).
var prunedStateErrSubstrings = []string{
	"trie node",
	"historical state",
	"archive mode",
	"archive node",
	"state not available",
	"state unavailable",
	"database is pruned",
	"storage trie",
}

// IsPrunedStateError reports whether err indicates the RPC node can't serve state for
// the requested historical block because it has pruned it (or isn't an archive node).
func IsPrunedStateError(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	for _, s := range prunedStateErrSubstrings {
		if strings.Contains(msg, s) {
			return true
		}
	}
	return false
}
