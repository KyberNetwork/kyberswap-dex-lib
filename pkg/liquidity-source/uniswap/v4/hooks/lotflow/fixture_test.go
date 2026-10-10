package lotflow

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const fixtureDir = "testdata/fixtures"

// loadAll loads every case listed in index.json, in index order.
func loadAll(t *testing.T) (*FixtureIndex, []*Case) {
	t.Helper()
	idx, err := LoadIndex(filepath.Join(fixtureDir, "index.json"))
	require.NoError(t, err)
	cases := make([]*Case, 0, len(idx.Cases))
	for _, e := range idx.Cases {
		c, err := LoadCase(filepath.Join(fixtureDir, filepath.Base(e.File)))
		require.NoError(t, err, e.ID)
		cases = append(cases, c)
	}
	return idx, cases
}

// A1 acceptance: all 85 fixtures parse into typed Cases, ids match index.json, 55 fill / 30 revert
// (the D-B regenerated set; F-FIXTURES.md §2).
func TestFixturesLoad(t *testing.T) {
	idx, cases := loadAll(t)
	require.Equal(t, FixtureSchema, idx.Schema)
	require.Equal(t, 85, idx.Count)
	require.Len(t, cases, 85)

	fills, reverts := 0, 0
	for i, c := range cases {
		e := idx.Cases[i]
		require.Equal(t, FixtureSchema, c.Schema, c.ID)
		require.Equal(t, e.ID, c.ID)
		require.Equal(t, e.Status, c.Outcome.Status, c.ID)
		require.Equal(t, e.AmountIn.String(), c.Input.AmountIn.String(), c.ID)
		require.Equal(t, e.AmountOut.String(), c.Outcome.AmountOut.String(), c.ID)

		// typed spot checks: the values every later stage depends on are present and parsed
		require.Positive(t, c.Pre.Nav.CheckedNavPerUnit18.Sign(), c.ID)
		require.Positive(t, c.Pin.BlockTimestamp.Sign(), c.ID)
		require.Equal(t, uint64(1_200_000), c.Pre.LegsConst.ProbeGas.Uint64(), c.ID)
		require.NotEmpty(t, c.Pre.Constituents, c.ID)
		for _, k := range c.Pre.Constituents {
			require.Positive(t, k.Unit.Sign(), c.ID)
			require.Equal(t, len(k.Candidates), int(k.CandidateCount), c.ID)
			for _, cand := range k.Candidates {
				require.Contains(t, []int{KindV4, KindV3}, cand.Kind, c.ID)
				require.Positive(t, cand.SqrtPriceX96.Sign(), c.ID)
				require.Equal(t, len(cand.Ticks), int(cand.TickCount), c.ID)
			}
		}

		switch c.Outcome.Status {
		case StatusOK:
			fills++
			require.True(t, c.IsFill())
			require.NotEmpty(t, c.Probes, c.ID)
			require.NotEmpty(t, c.Derived.Legs, c.ID)
			require.Len(t, c.Derived.Legs, len(e.Legs), c.ID)
		case StatusRevert:
			reverts++
			require.NotNil(t, c.Derived.RevertInnermost, c.ID)
			require.Equal(t, e.Revert, c.Derived.RevertInnermost.Name, c.ID)
		default:
			t.Fatalf("%s: unknown status %q", c.ID, c.Outcome.Status)
		}
	}
	require.Equal(t, 55, fills)
	require.Equal(t, 30, reverts)
}
