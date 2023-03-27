package router

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/abis"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/types"
)

func BuildAndPackSwapSimpleModeInputs(
	executorAddress string,
	executorData []byte,
	data types.EncodingData,
) ([]byte, error) {
	swapInputs := SwapSimpleModeInputs{
		Caller:       common.HexToAddress(executorAddress),
		Desc:         buildSwapDescriptionV2ForSwapSimpleMode(data),
		ExecutorData: executorData,
		ClientData:   data.ClientData,
	}

	return PackSwapSimpleModeInputs(swapInputs)
}

func PackSwapSimpleModeInputs(swapSimpleModeInputs SwapSimpleModeInputs) ([]byte, error) {
	return abis.MetaAggregationRouterV2.Pack(
		MethodNameSwapSimpleMode,
		swapSimpleModeInputs.Caller,
		swapSimpleModeInputs.Desc,
		swapSimpleModeInputs.ExecutorData,
		swapSimpleModeInputs.ClientData,
	)
}

func UnpackSwapSimpleModeInputs(data []byte) (SwapSimpleModeInputs, error) {
	method, err := abis.MetaAggregationRouterV2.MethodById(data)
	if err != nil {
		return SwapSimpleModeInputs{}, err
	}

	unpacked, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return SwapSimpleModeInputs{}, err
	}

	var inputs SwapSimpleModeInputs
	if err = method.Inputs.Copy(&inputs, unpacked); err != nil {
		return SwapSimpleModeInputs{}, err
	}

	return inputs, nil
}
