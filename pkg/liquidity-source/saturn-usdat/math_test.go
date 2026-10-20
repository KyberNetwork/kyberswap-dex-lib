package saturnusdat

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func u(s string) *uint256.Int { return uint256.MustFromDecimal(s) }

func TestConvertAmounts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		from, to uint8
		in, out  string
		err      error
	}{
		{"same", 6, 6, "123456789", "123456789", nil},
		{"up", 6, 18, "1500000", "1500000000000000000", nil},
		{"down floors", 18, 6, "1999999999999", "1", nil},
		{"down to zero", 18, 6, "999999999999", "0", nil},
		{"down huge diff", 255, 0, "5", "0", nil},
		{"up overflow", 0, 77, "11", "", ErrOverflow},
		{"up huge diff", 0, 100, "1", "", ErrOverflow},
		{"up max fits", 0, 18, "115792089237316195423570985008687907853269984665640564039457", "115792089237316195423570985008687907853269984665640564039457000000000000000000", nil},
		{"up max + 1 overflow", 0, 18, "115792089237316195423570985008687907853269984665640564039458", "", ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var res uint256.Int
			err := convertAmounts(&res, tt.from, tt.to, u(tt.in))
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.out, res.Dec())
		})
	}
}

func TestWrapAmounts(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name             string
		aDec, eDec       uint8
		amount, cap, bal string
		ext, eff         string
		err              error
	}{
		{"1:1", 6, 6, "1000000000", "10000000000000", "0", "1000000000", "1000000000", nil},
		{"exactly at capacity", 6, 6, "10000000000000", "10000000000000", "0", "10000000000000", "10000000000000", nil},
		{"capacity + 1", 6, 6, "10000000000001", "10000000000000", "0", "", "", ErrAssetCapReached},
		{"balance + amount == cap", 6, 6, "9999000000000", "10000000000000", "1000000000", "9999000000000", "9999000000000", nil},
		{"balance + amount == cap + 1", 6, 6, "9999000000001", "10000000000000", "1000000000", "", "", ErrAssetCapReached},
		{"balance above cap", 6, 6, "1", "10", "11", "", "", ErrAssetCapReached},
		{"zero amount", 6, 6, "0", "10000000000000", "0", "", "", ErrZeroAmount},
		{"dust: asset 18 dec", 18, 6, "1999999999999", "10000000000000000000000", "0", "1", "1000000000000", nil},
		{"dust truncates to zero", 18, 6, "999999999999", "10000000000000000000000", "0", "", "", ErrZeroAmount},
		{"asset 8 dec", 8, 6, "12345678", "10000000000000", "0", "123456", "12345600", nil},
		{"convert overflow", 0, 77, "11", "10000000000000", "0", "", "", ErrOverflow},
		{"balance at uint240 max with uint256 cap", 6, 6, "1", "115792089237316195423570985008687907853269984665640564039457584007913129639935", "1766847064778384329583297500742918515827483896875618958121606201292619775", "", "", ErrOverflow},
		{"balance fills uint240 exactly", 6, 6, "1", "115792089237316195423570985008687907853269984665640564039457584007913129639935", "1766847064778384329583297500742918515827483896875618958121606201292619774", "1", "1", nil},
		{"balance add overflow", 6, 6, "1", "10", "115792089237316195423570985008687907853269984665640564039457584007913129639935", "", "", ErrOverflow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ext, eff, err := wrapAmounts(tt.aDec, tt.eDec, u(tt.amount), u(tt.cap), u(tt.bal))
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.ext, ext.Dec())
			assert.Equal(t, tt.eff, eff.Dec())
		})
	}
}

func TestRemainingCapacity(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "7", func() string { r := remainingCapacity(u("10"), u("3")); return r.Dec() }())
	assert.Equal(t, "0", func() string { r := remainingCapacity(u("10"), u("10")); return r.Dec() }())
	assert.Equal(t, "0", func() string { r := remainingCapacity(u("10"), u("11")); return r.Dec() }())
}
