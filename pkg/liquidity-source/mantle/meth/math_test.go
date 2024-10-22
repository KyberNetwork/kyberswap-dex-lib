package meth

import (
	"testing"

	"github.com/KyberNetwork/ethrpc"
	"github.com/holiman/uint256"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MathTestSuite struct {
	suite.Suite

	client  *ethrpc.Client
	tracker PoolTracker
}

func (ts *MathTestSuite) TestMulDiv() {
	x := uint256.MustFromDecimal("20000000000000000")
	y := uint256.MustFromDecimal("4692604041539924395219916232")
	denominator := uint256.MustFromDecimal("4913213212083834958451170000")

	result, err := mulDiv(x, y, denominator)
	require.NoError(ts.T(), err)
	require.Equal(ts.T(), "19101975994034486", result.String())
}

func TestMulDivTestSuite(t *testing.T) {
	// t.Skip("Skipping testing in CI environment")
	suite.Run(t, new(MathTestSuite))
}
