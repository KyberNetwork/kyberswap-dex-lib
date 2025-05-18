package balancerv1

import (
	"testing"

	"github.com/holiman/uint256"
	"github.com/stretchr/testify/assert"
)

func TestBPowApprox(t *testing.T) {
	t.Parallel()
	res, err := BNum.BPowApprox(
		uint256.MustFromDecimal("999999998494616385"),
		uint256.MustFromDecimal("750000000000000000"),
		uint256.MustFromDecimal("100000000"),
	)
	assert.Nil(t, err)
	assert.Equal(t, "999999998870962289", res.Dec())
}
