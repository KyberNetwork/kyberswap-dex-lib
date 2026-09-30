package titan

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
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

func pammEntry(account, slot, value string) json.RawMessage {
	return json.RawMessage(`{"stateOverride":{"` + account + `":{"balance":"0x0","nonce":"0x1","stateDiff":{"` +
		slot + `":"` + value + `"}}}}`)
}

// TestMergeResponses: a pAMM entry only one region carries must survive the
// merge (Stelaxis showed up in ~1/3 of single-region fetches). Entries from a
// region lagging a block are stale and dropped, and an entry present in two
// regions is taken whole from the first, never mixed slot-by-slot.
func TestMergeResponses(t *testing.T) {
	t.Parallel()

	a, b, c := "0x000000000000000000000000000000000000000a", "0x000000000000000000000000000000000000000b",
		"0x000000000000000000000000000000000000000c"
	merged := mergeResponses([]map[string]json.RawMessage{
		{"blockNumber": json.RawMessage(`"0x10"`), "pammA": pammEntry(a, "0x1", "0x11")},
		{"blockNumber": json.RawMessage(`"0x10"`), "pammA": pammEntry(a, "0x2", "0x22"), "pammB": pammEntry(b, "0x1", "0x33")},
		{"blockNumber": json.RawMessage(`"0x0f"`), "pammC": pammEntry(c, "0x1", "0x44")},
	})

	overrides := mergeOverrides(merged)
	if len(overrides) != 2 {
		t.Fatalf("want accounts a and b only, got %d accounts", len(overrides))
	}
	if diff := overrides[common.HexToAddress(a)].StateDiff; len(diff) != 1 || diff[common.HexToHash("0x1")] != common.HexToHash("0x11") {
		t.Fatalf("pammA must come whole from the first response, got %v", diff)
	}
	if _, ok := overrides[common.HexToAddress(b)]; !ok {
		t.Fatalf("pammB, carried only by the second region, must be merged in")
	}
	if _, ok := overrides[common.HexToAddress(c)]; ok {
		t.Fatalf("pammC comes from an older block and must be dropped")
	}
}

func titanServer(t *testing.T, delay time.Duration, entries map[string]json.RawMessage) *rpc.Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
		result := map[string]json.RawMessage{"blockNumber": json.RawMessage(`"0x10"`), "slot": json.RawMessage(`100`)}
		for k, v := range entries {
			result[k] = v
		}
		body, _ := json.Marshal(result)
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":%s}`, req.ID, body)
	}))
	t.Cleanup(srv.Close)
	c, err := rpc.Dial(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(c.Close)
	return c
}

// TestFetchState_MergeWindow: a region answering shortly after the first
// contributes its entries, and a region slower than the merge window must not
// hold up the tracker cycle (pool-service refreshes titan-prop every 3s).
func TestFetchState_MergeWindow(t *testing.T) {
	t.Parallel()

	a, b, c := "0x000000000000000000000000000000000000000a", "0x000000000000000000000000000000000000000b",
		"0x000000000000000000000000000000000000000c"
	clients := []*rpc.Client{
		titanServer(t, 0, map[string]json.RawMessage{"pammA": pammEntry(a, "0x1", "0x11")}),
		titanServer(t, 50*time.Millisecond, map[string]json.RawMessage{"pammB": pammEntry(b, "0x1", "0x22")}),
		titanServer(t, 5*time.Second, map[string]json.RawMessage{"pammC": pammEntry(c, "0x1", "0x33")}),
	}

	start := time.Now()
	state := FetchState(context.Background(), clients, 10*time.Second)
	if elapsed := time.Since(start); elapsed > DefaultMergeWindow+time.Second {
		t.Fatalf("FetchState waited %v for the slow region; must return after the merge window", elapsed)
	}
	if _, ok := state.Overrides[common.HexToAddress(a)]; !ok {
		t.Fatalf("first region's entry missing")
	}
	if _, ok := state.Overrides[common.HexToAddress(b)]; !ok {
		t.Fatalf("entry from the region answering within the merge window missing")
	}
	if _, ok := state.Overrides[common.HexToAddress(c)]; ok {
		t.Fatalf("entry from the region slower than the merge window must not be merged")
	}
	if state.BlockTimestamp != beaconGenesisTS+100*secsPerSlot {
		t.Fatalf("BlockTimestamp must come from the merged slot, got %d", state.BlockTimestamp)
	}
}
