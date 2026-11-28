package efficientprop

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// The fixtures are trimmed live answers of the two methods (two venues, two entries). The venues answer types its
// blockNumber as a JSON number, the old answer as a hex string; both slots are numbers.
func TestParseAnswersLiveFixtures(t *testing.T) {
	load := func(name string) map[string]json.RawMessage {
		b, err := os.ReadFile("testdata/" + name)
		require.NoError(t, err)
		var m map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(b, &m))
		return m
	}
	type diffs = map[string]struct {
		StateDiff map[string]string `json:"stateDiff"`
	}
	check := func(name string, want int, first func(m map[string]json.RawMessage) (string, diffs)) {
		m := load(name)
		got, skipped := parseAnswers(0, m)
		require.Len(t, got, 1, name)
		assert.Empty(t, skipped, name)
		a := got[0]
		require.NotNil(t, a.number, name)
		assert.Equal(t, uint64(26511070), a.number.Uint64(), name)
		assert.Equal(t, uint64(15751079), a.slot, name)
		assert.Len(t, a.accounts, want, name)
		addr, d := first(m)
		for acct, v := range d {
			if len(v.StateDiff) == 0 {
				continue
			}
			o := a.accounts[common.HexToAddress(acct)]
			require.NotEmpty(t, o.StateDiff, "%s: %s of %s", name, acct, addr)
			for k, w := range v.StateDiff {
				assert.Equal(t, common.HexToHash(w), o.StateDiff[common.HexToHash(k)], name)
			}
			return
		}
		t.Fatalf("%s: no account with a stateDiff", name)
	}
	check("titan_venues_live.json", 2, func(m map[string]json.RawMessage) (string, diffs) {
		var v []struct {
			Quote string `json:"quoteAddress"`
			SO    diffs  `json:"stateOverride"`
		}
		require.NoError(t, json.Unmarshal(m["venues"], &v))
		return v[0].Quote, v[0].SO
	})
	check("titan_state_overrides_live.json", 1, func(m map[string]json.RawMessage) (string, diffs) {
		for k, raw := range m {
			var e struct {
				SO diffs `json:"stateOverride"`
			}
			if k != "slot" && k != "blockNumber" && json.Unmarshal(raw, &e) == nil {
				return k, e.SO
			}
		}
		return "", nil
	})
}
