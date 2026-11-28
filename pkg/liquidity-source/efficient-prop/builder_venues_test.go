package efficientprop

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// venueJSON is one entry of a titan_getPammVenues answer, as the takers docs show it: its own block, slot and
// stateOverride beside the maker's identity fields.
func venueJSON(cell common.Address, block string, slot uint64, word string) map[string]any {
	return map[string]any{
		"maker": "0x4f8dd270acE370dE2f6Ac5EBf3230cBE099a0753", "quoteAddress": cell.Hex(),
		"pricedAddresses": []string{}, "assetPairs": []string{}, "slot": slot, "blockNumber": block,
		"stateOverride": map[string]any{cell.Hex(): map[string]any{
			"stateDiff": map[string]string{quoteKey(0).Hex(): word}, "balance": "0x0", "nonce": "0x1"}},
	}
}

func rawMap(t *testing.T, v any) map[string]json.RawMessage {
	t.Helper()
	b, err := json.Marshal(v)
	require.NoError(t, err)
	var m map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

const w1, w2, w3 = "0x0000000000000000000000000000000000000000000000000000000000000001",
	"0x0000000000000000000000000000000000000000000000000000000000000002",
	"0x0000000000000000000000000000000000000000000000000000000000000003"

// parseAnswers reads both shapes of a /state result into answers, the venues shape one answer a block, and counts
// each venue it skips by reason.
func TestParseAnswers(t *testing.T) {
	a, b := cellOf(1), cellOf(2)
	oldShape := rawMap(t, map[string]any{"slot": 7, "blockNumber": "0x4d3",
		"entry": map[string]any{"stateOverride": map[string]any{a.Hex(): map[string]any{
			"stateDiff": map[string]string{quoteKey(0).Hex(): w1}, "balance": "0x0", "nonce": "0x1"}}}})
	for _, c := range []struct {
		name    string
		result  map[string]json.RawMessage
		blocks  []uint64 // the answers' blocks, highest first
		cells   [][]common.Address
		skipped map[string]int
	}{
		{"old shape", oldShape, []uint64{0x4d3}, [][]common.Address{{a}}, nil},
		{"new shape, one block", rawMap(t, map[string]any{"venues": []any{
			venueJSON(a, "0x4d3", 7, w1), venueJSON(b, "0x4d3", 7, w2)}}),
			[]uint64{0x4d3}, [][]common.Address{{a, b}}, nil},
		{"mixed blocks", rawMap(t, map[string]any{"venues": []any{
			venueJSON(a, "0x4d2", 6, w1), venueJSON(b, "0x4d3", 7, w2), venueJSON(a, "0x4d3", 7, w3)}}),
			[]uint64{0x4d3, 0x4d2}, [][]common.Address{{a, b}, {a}}, nil},
		{"an empty venues list", rawMap(t, map[string]any{"venues": []any{}}), nil, nil, nil},
		{"a malformed venue is skipped and counted", rawMap(t, map[string]any{"venues": []any{
			venueJSON(a, "0x4d3", 7, w1), "not an object", 5,
			map[string]any{"slot": 7, "stateOverride": map[string]any{}},
			map[string]any{"slot": 7, "blockNumber": "0x4d3"},
			map[string]any{"slot": 7, "blockNumber": "0x4d3", "stateOverride": "x"},
			map[string]any{"slot": 7, "blockNumber": "zz", "stateOverride": map[string]any{}}}}),
			[]uint64{0x4d3}, [][]common.Address{{a}},
			map[string]int{"venue is not an object": 2, "venue has no block number": 2,
				"venue has no state override": 1, "venue state override unreadable": 1}},
		{"venues that is not a list is skipped", rawMap(t, map[string]any{"venues": map[string]any{}}), nil, nil,
			map[string]int{"venues is not a list": 1}},
	} {
		got, skipped := parseAnswers(3, c.result)
		require.Len(t, got, len(c.blocks), c.name)
		assert.Equal(t, c.skipped, skipped, c.name)
		for k, an := range got {
			assert.Equal(t, 3, an.rank, c.name)
			require.NotNil(t, an.number, c.name)
			assert.Equal(t, c.blocks[k], an.number.Uint64(), c.name)
			assert.Len(t, an.accounts, len(c.cells[k]), c.name)
			for _, cell := range c.cells[k] {
				assert.Contains(t, an.accounts, cell, c.name)
			}
			assert.False(t, an.malformed, c.name)
		}
	}
	// the highest block's answer keeps its venues whatever the list's order
	got, _ := parseAnswers(0, rawMap(t, map[string]any{"venues": []any{
		venueJSON(a, "0x4d3", 7, w1), venueJSON(b, "0x4d2", 6, w2), venueJSON(b, "0x4d3", 7, w3)}}))
	require.Len(t, got, 2)
	assert.Len(t, got[0].accounts, 2)
	assert.Equal(t, uint64(7), got[0].slot)
	assert.Equal(t, beaconGenesis+slotSeconds*7, int(got[0].time))
}

func dialClients(t *testing.T, urls ...string) []stateClient {
	var cs []stateClient
	for _, u := range urls {
		c, err := rpc.DialContext(context.Background(), u)
		require.NoError(t, err)
		t.Cleanup(c.Close)
		cs = append(cs, stateClient{url: u, client: c})
	}
	return cs
}

// With the new method configured the request names it, and its venues answer is read.
func TestFetchAnswersAsksTheConfiguredMethod(t *testing.T) {
	cell := cellOf(1)
	var seen string
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		_ = json.NewDecoder(r.Body).Decode(&msg)
		seen = msg.Method
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": msg.ID,
			"result": map[string]any{"venues": []any{venueJSON(cell, "0x4d3", 7, w1)}}})
	}))
	t.Cleanup(s.Close)
	cs := dialClients(t, s.URL)
	cs[0].method = methodVenues
	got := fetchAnswers(context.Background(), cs, time.Second, 0, nil)
	assert.Equal(t, methodVenues, seen)
	require.Len(t, got, 1)
	assert.Contains(t, got[0].accounts, cell)
	cs[0].method = ""
	fetchAnswers(context.Background(), cs, time.Second, 0, nil)
	assert.Equal(t, stateMethod, seen, "the old method stays the default")
}

func TestConfigValidatesStateMethod(t *testing.T) {
	cfg := ethConfig("http://x")
	for m, ok := range map[string]bool{"": true, stateMethod: true, methodVenues: true, "titan_other": false} {
		cfg.Fronts[0].StateMethod = m
		err := cfg.validate()
		assert.Equal(t, ok, err == nil, "%q: %v", m, err)
	}
}
