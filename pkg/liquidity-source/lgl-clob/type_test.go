package lglclob

import (
	"testing"

	"github.com/goccy/go-json"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hanji's API sends the fee as a number and xpress's as a string, and both
// must read as a market rather than fail the list.
func TestSwapFee(t *testing.T) {
	t.Parallel()

	for raw, want := range map[string]float64{
		`{"aggressiveFee":0.0001}`:   0.0001,
		`{"aggressiveFee":"0.0003"}`: 0.0003,
		`{}`:                         0,
	} {
		var market MarketInfo
		require.NoError(t, json.Unmarshal([]byte(raw), &market), raw)
		assert.Equal(t, want, swapFee(&market), raw)
	}
}
