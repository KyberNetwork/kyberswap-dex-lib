package router

import (
	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
)

const MethodNameSwapCompressed = "swapCompressed"

func PackSwapInputs(
	executorAddress string,
	executorData []byte,
	data types.EncodingData,
) ([]byte, error) {
	swapInputs := SwapExecutionParams{
		CallTarget:    common.HexToAddress(executorAddress),
		ApproveTarget: common.HexToAddress(valueobject.ZeroAddress),
		TargetData:    executorData,
		Desc:          buildSwapDescriptionV2ForSwap(executorAddress, data),
		ClientData:    data.ClientData,
	}

	compressedData, err := packNormalSwapData(swapInputs)
	if err != nil {
		return nil, err
	}

	return abis.MetaAggregationRouterV2Optimistic.Pack(
		MethodNameSwapCompressed,
		compressedData,
	)
}

func UnpackNormalSwapData(data []byte) (SwapExecutionParams, error) {
	method, err := abis.MetaAggregationRouterV2Optimistic.MethodById(data)
	if err != nil {
		return SwapExecutionParams{}, err
	}

	unpacked, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return SwapExecutionParams{}, err
	}

	var inputs SwapDataCompress
	if err = method.Inputs.Copy(&inputs, unpacked); err != nil {
		return SwapExecutionParams{}, err
	}

	return unpackNormalSwapData(inputs.Data)
}

func packNormalSwapData(swapInputs SwapExecutionParams) ([]byte, error) {
	desc, err := packSwapDescriptionV2(swapInputs.Desc)
	if err != nil {
		return nil, err
	}

	return pack.Pack(
		swapInputs.CallTarget,
		swapInputs.ApproveTarget,
		swapInputs.TargetData,
		pack.RawBytes(desc),
		swapInputs.ClientData,
	)
}

func unpackNormalSwapData(data []byte) (SwapExecutionParams, error) {
	var inputs SwapExecutionParams
	var startByte int

	inputs.CallTarget, startByte = pack.ReadAddress(data, startByte)
	inputs.ApproveTarget, startByte = pack.ReadAddress(data, startByte)
	inputs.TargetData, startByte = pack.ReadBytes(data, startByte)
	inputs.Desc, startByte = unpackSwapDescriptionV2(data, startByte)
	inputs.ClientData, _ = pack.ReadBytes(data, startByte)

	return inputs, nil
}
