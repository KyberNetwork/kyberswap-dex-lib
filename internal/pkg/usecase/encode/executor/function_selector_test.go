package executor

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func TestGenMethodID(t *testing.T) {
	testCases := []struct {
		function   FunctionSelector
		expectedId string
	}{
		{expectedId: "0x59361199", function: FunctionSelectorUniswap},
		{expectedId: "0xa3722546", function: FunctionSelectorKSClassic},
		{expectedId: "0x55fad2fb", function: FunctionSelectorVelodrome},
		{expectedId: "0xa9b3e398", function: FunctionSelectorFraxSwap},
		{expectedId: "0x8df4a16b", function: FunctionSelectorCamelotSwap},
		{expectedId: "0xa8d2cb11", function: FunctionSelectorStableSwap},
		{expectedId: "0xd90ce491", function: FunctionSelectorCurveSwap},
		{expectedId: "0x63407a49", function: FunctionSelectorUniV3KSElastic},
		{expectedId: "0x8cc7a56b", function: FunctionSelectorBalancerV2},
		{expectedId: "0x7b797563", function: FunctionSelectorDODO},
		{expectedId: "0x3b284cfe", function: FunctionSelectorGMX},
		{expectedId: "0x74836acb", function: FunctionSelectorSynthetix},
		{expectedId: "0x0ca8ebf1", function: FunctionSelectorWSTETH},
		{expectedId: "0x92749fe1", function: FunctionSelectorPlatypus},
		{expectedId: "0x8f079854", function: FunctionSelectorPSM},
		{expectedId: "0xd6984a6d", function: FunctionSelectorLimitOrder},
	}

	for idx, tc := range testCases {
		t.Run(fmt.Sprintf("it should gen method id correctly %d", idx), func(t *testing.T) {
			assert.EqualValues(t, common.HexToHash(tc.expectedId), common.BytesToHash(tc.function.ID[:]))
		})
	}
}
