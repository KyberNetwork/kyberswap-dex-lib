package executor

import (
	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/KyberNetwork/router-service/internal/pkg/constant/abitypes"
)

var (
	CallBytesInputsABIArguments          abi.Arguments
	SwapSingleSequenceInputsABIArguments abi.Arguments
	SimpleSwapDataABIArguments           abi.Arguments
	SwapExecutorDescriptionABIType       abi.Type
	SwapSequenceABIType                  abi.Type
)

func init() {
	SwapExecutorDescriptionABIType, _ = abi.NewType(
		"tuple",
		"struct AggregationExecutor.SwapExecutorDescription",
		[]abi.ArgumentMarshaling{
			{
				Name: "swapSequences", Type: "tuple[][]",
				Components: []abi.ArgumentMarshaling{
					{Name: "data", Type: "bytes"},
					{Name: "functionSelector", Type: "bytes4"},
				},
			},
			{Name: "tokenIn", Type: "address"},
			{Name: "tokenOut", Type: "address"},
			{Name: "minTotalAmountOut", Type: "uint256"},
			{Name: "to", Type: "address"},
			{Name: "deadline", Type: "uint256"},
			{Name: "destTokenFeeData", Type: "bytes"},
		},
	)

	SwapSequenceABIType, _ = abi.NewType(
		"tuple[]",
		"",
		[]abi.ArgumentMarshaling{
			{Name: "data", Type: "bytes"},
			{Name: "functionSelector", Type: "bytes4"},
		},
	)

	SwapSingleSequenceInputsABIArguments = abi.Arguments{
		{Name: "swapData", Type: SwapSequenceABIType},
	}

	CallBytesInputsABIArguments = abi.Arguments{
		{Name: "data", Type: SwapExecutorDescriptionABIType},
	}

	SimpleSwapDataABIArguments = abi.Arguments{
		{Name: "firstPools", Type: abitypes.AddressArr},
		{Name: "firstSwapAmounts", Type: abitypes.Uint256Arr},
		{Name: "swapDatas", Type: abitypes.BytesArr},
		{Name: "deadline", Type: abitypes.Uint256},
		{Name: "destTokenFeeData", Type: abitypes.Bytes},
	}
}
