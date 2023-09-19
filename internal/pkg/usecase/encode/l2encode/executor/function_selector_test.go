package executor

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestGenMethodID(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		function   FunctionSelector
		expectedId string
	}{
		{expectedId: "0x8f7ef8c1", function: FunctionSelectorUniswap},
		{expectedId: "0x3775c3bb", function: FunctionSelectorKSClassic},
		{expectedId: "0x4cf50733", function: FunctionSelectorVelodrome},
		{expectedId: "0x2fe04c4d", function: FunctionSelectorFraxSwap},
		{expectedId: "0x7157338b", function: FunctionSelectorLimitOrder},
		{expectedId: "0x4a0bc786", function: FunctionSelectorSynthetix},
		{expectedId: "0x8f3af853", function: FunctionSelectorUniV3KSElastic},
		{expectedId: "0x977346ca", function: FunctionSelectorGMX},
		{expectedId: "0xc12d7767", function: FunctionSelectorStableSwap},
		{expectedId: "0x81a9195d", function: FunctionSelectorCurveSwap},
		{expectedId: "0x5d0d2501", function: FunctionSelectorBalancerV2},
		{expectedId: "0xca554d0e", function: FunctionSelectorDODO},
		{expectedId: "0x4981e7f5", function: FunctionSelectorCamelotSwap},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("it should gen method id correctly %d", idx), func(t *testing.T) {
			assert.EqualValues(t, common.HexToHash(tc.expectedId), common.BytesToHash(tc.function.ID[:]))
		})
	}
}
