package nemoprop

import (
	"math"
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
)

func TestShapeLadder(t *testing.T) {
	t.Parallel()
	raw := []ladder.Point{{100, 200}, {200, 400}, {400, 700}, {800, 700}}

	tests := map[string]struct {
		cap    uint64
		buffer int64
		want   []ladder.Point
	}{
		"below cap untouched":  {cap: 1000, want: raw},
		"flat cap is cut":      {cap: 700, want: raw[:3]},
		"boundary on chord":    {cap: 550, want: []ladder.Point{{100, 200}, {200, 400}, {300, 550}}},
		"cap below first":      {cap: 100, want: []ladder.Point{{50, 100}}},
		"nothing deliverable":  {cap: 0, want: nil},
		"buffer after cutting": {cap: 550, buffer: 9000, want: []ladder.Point{{100, 180}, {200, 360}, {300, 495}}},
		"buffer >= 1 ignored":  {cap: 1000, buffer: 10000, want: raw},
	}
	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			before := append([]ladder.Point(nil), raw...)
			got := shapeLadder(raw, uint256.NewInt(tc.cap), tc.buffer)
			assert.Equal(t, len(tc.want), len(got))
			for i := range tc.want {
				assert.InDelta(t, tc.want[i][0], got[i][0], 1e-9)
				assert.InDelta(t, tc.want[i][1], got[i][1], 1e-9)
			}
			assert.Equal(t, before, raw, "raw ladder must not be modified")
		})
	}
}

func TestShapeLadders_DirectionUsesOutputReserve(t *testing.T) {
	t.Parallel()
	raw := [2][]ladder.Point{{{100, 100}, {200, 200}}, {{100, 100}, {200, 200}}}
	got := shapeLadders(raw, [2]*uint256.Int{uint256.NewInt(150), uint256.NewInt(1000)}, 0)
	assert.Len(t, got[0], 2, "base -> market pays out the market token (reserve 1)")
	assert.Equal(t, []ladder.Point{{100, 100}, {150, 150}}, got[1], "market -> base pays out base (reserve 0)")
}

func TestValidateLadder(t *testing.T) {
	t.Parallel()
	assert.NoError(t, validateLadder(nil))
	assert.NoError(t, validateLadder([]ladder.Point{{1, 2}, {2, 2}, {3, 5}}))
	for name, l := range map[string][]ladder.Point{
		"non-increasing input": {{2, 2}, {2, 3}},
		"decreasing output":    {{1, 3}, {2, 2}},
		"zero output":          {{1, 0}},
		"negative input":       {{-1, 1}},
		"nan":                  {{math.NaN(), 1}},
		"inf":                  {{1, math.Inf(1)}},
	} {
		assert.ErrorIs(t, validateLadder(l), errInvalidLadder, name)
	}
}

func TestDeliverable(t *testing.T) {
	t.Parallel()
	assert.Equal(t, uint256.NewInt(5), deliverable(uint256.NewInt(5), uint256.NewInt(9)))
	assert.Equal(t, uint256.NewInt(4), deliverable(uint256.NewInt(5), uint256.NewInt(4)))
}
