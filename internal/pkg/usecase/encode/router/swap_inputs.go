package router

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
)

func BuildAndPackSwapInputs(
	executorAddress string,
	executorData []byte,
	data types.EncodingData,
) ([]byte, error) {
	swapInputs := SwapInputs{
		Execution: SwapExecutionParams{
			CallTarget:    common.HexToAddress(executorAddress),
			ApproveTarget: common.HexToAddress(constant.AddressZero),
			Desc:          buildSwapDescriptionV2ForSwap(executorAddress, data),
			TargetData:    executorData,
			ClientData:    data.ClientData,
		},
	}

	return PackSwapInputs(swapInputs)
}

func PackSwapInputs(swapInputs SwapInputs) ([]byte, error) {
	return abis.MetaAggregationRouterV2.Pack(
		MethodNameSwap,
		swapInputs.Execution,
	)
}

func UnpackSwapInputs(data []byte) (SwapInputs, error) {
	method, err := abis.MetaAggregationRouterV2.MethodById(data)
	if err != nil {
		return SwapInputs{}, err
	}

	unpacked, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return SwapInputs{}, err
	}

	var inputs SwapInputs
	if err = method.Inputs.Copy(&inputs, unpacked); err != nil {
		return SwapInputs{}, err
	}

	return inputs, nil
}
