package titan

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
)

// TestMergeOverrides locks in the "flatten everything, no per-venue lookup"
// behavior FetchState relies on: every pAMM entry in a
// titan_getPammStateOverrides response gets merged into one combined
// account -> stateDiff map (skipping the "slot"/"blockNumber" metadata keys),
// regardless of which top-level key -- venue address or a diverging "top-level
// stream address" like bopAMM's -- delivered it. See FetchState's doc comment
// for the rationale.
func TestMergeOverrides(t *testing.T) {
	t.Parallel()

	registry := "0xda7afeed01fe625cf15d187a19f94b45f00b8c5f"

	// Mirrors a real titan_getPammStateOverrides response: two pAMM entries
	// (a venue that streams under its own address, and one -- like bopAMM --
	// that streams under a different "top-level stream address") each
	// touching different slots of the SAME shared registry contract.
	raw := map[string]json.RawMessage{
		"slot":        json.RawMessage(`14984765`),
		"blockNumber": json.RawMessage(`"0x188df0d"`),
		// Tempest, streaming under its own venue address.
		"0x00000003f1ec2379e79f58e12ec6c4f51ee92149": json.RawMessage(`{
			"stateOverride": {
				"` + registry + `": {
					"balance": "0x0", "nonce": "0x1",
					"stateDiff": {"0xaaaa": "0x1111"}
				}
			}
		}`),
		// bopAMM, streaming under a "top-level stream address" that differs
		// from its own venue/call-target address -- but touching the same
		// registry contract, at a different slot.
		"0xb0999914b3de1be58ef2416af09bd2e7f8aad03c": json.RawMessage(`{
			"stateOverride": {
				"` + registry + `": {
					"balance": "0x0", "nonce": "0x1",
					"stateDiff": {"0xbbbb": "0x2222"}
				}
			}
		}`),
	}

	merged := mergeOverrides(raw)

	if len(merged) != 1 {
		t.Fatalf("expected exactly 1 merged account (the shared registry), got %d", len(merged))
	}
	acct, ok := merged[common.HexToAddress(registry)]
	if !ok {
		t.Fatalf("expected the registry contract to be present in the merged overrides")
	}
	// Both entries' slots must be present -- this is the "flatten everything"
	// property: a caller quoting bopAMM (which streams under a different
	// top-level key) still gets Tempest's slot merged in too, and vice versa,
	// which is fine since each pAMM's own slots are what its quote() actually
	// reads -- unrelated slots are simply ignored by the venue being quoted.
	if len(acct.StateDiff) != 2 {
		t.Fatalf("expected 2 merged slots (one per pAMM entry), got %d", len(acct.StateDiff))
	}
	if got := acct.StateDiff[common.HexToHash("0xaaaa")]; got != common.HexToHash("0x1111") {
		t.Fatalf("Tempest's slot missing or wrong: got %s", got)
	}
	if got := acct.StateDiff[common.HexToHash("0xbbbb")]; got != common.HexToHash("0x2222") {
		t.Fatalf("bopAMM's slot missing or wrong: got %s", got)
	}

	// "slot" and "blockNumber" metadata keys must never be treated as pAMM
	// entries (they're not addresses and have no stateOverride).
	if _, ok := merged[common.HexToAddress("slot")]; ok {
		t.Fatalf("metadata key \"slot\" must not leak into merged overrides")
	}
}

func TestMergeOverrides_Empty(t *testing.T) {
	t.Parallel()

	if got := mergeOverrides(nil); len(got) != 0 {
		t.Fatalf("expected empty map for nil input, got %v", got)
	}
	if got := mergeOverrides(map[string]json.RawMessage{
		"slot": json.RawMessage(`123`),
	}); len(got) != 0 {
		t.Fatalf("expected empty map when only metadata keys present, got %v", got)
	}
}
