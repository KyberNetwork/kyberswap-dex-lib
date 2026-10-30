package ephemeral

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckBins(t *testing.T) {
	bin := func(id uint32, x, y int64) Bin { return Bin{ID: id, ReserveX: big.NewInt(x), ReserveY: big.NewInt(y)} }
	bins := []Bin{bin(10, 0, 5), bin(11, 3, 2), bin(12, 4, 0)}

	assert.NoError(t, CheckBins(bins, big.NewInt(7), big.NewInt(7)))
	assert.NoError(t, CheckBins(nil, big.NewInt(0), big.NewInt(0)), "a pair without liquidity")
	assert.NoError(t, CheckBins(bins, nil, nil), "a pool that keeps no reserves")
	for name, c := range map[string]struct {
		bins []Bin
		x, y int64
	}{
		"a bin missing":   {bins[:2], 7, 7},
		"reserve Y off":   {bins, 7, 8},
		"out of order":    {[]Bin{bins[1], bins[0], bins[2]}, 7, 7},
		"a bin repeated":  {[]Bin{bins[0], bins[0], bins[1], bins[2]}, 7, 12},
		"an empty bin":    {append([]Bin{bin(9, 0, 0)}, bins...), 7, 7},
		"reserves absent": {bins, 0, 0},
	} {
		assert.ErrorIs(t, CheckBins(c.bins, big.NewInt(c.x), big.NewInt(c.y)), ErrBinsInconsistent, name)
	}
	for name, bins := range map[string][]Bin{
		"out of order": {bin(5, 1, 0), bin(1, 0, 1)},
		"repeated":     {bin(5, 1, 0), bin(5, 0, 1)},
		"empty":        {bin(5, 0, 0)},
	} {
		assert.ErrorIs(t, CheckBins(bins, nil, nil), ErrBinsInconsistent, "no reserves, "+name)
	}
}
