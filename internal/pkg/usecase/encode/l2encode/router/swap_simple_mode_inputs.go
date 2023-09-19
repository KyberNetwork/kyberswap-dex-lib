package router

import (
	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/ethereum/go-ethereum/common"
)

const MethodNameSwapSimpleModeCompressed = "swapSimpleModeCompressed"

func PackSwapSimpleModeInputs(
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

	compressedData, err := packSwapSimpleModeInputs(swapInputs)
	if err != nil {
		return nil, err
	}

	return abis.MetaAggregationRouterV2Optimistic.Pack(
		MethodNameSwapSimpleModeCompressed,
		compressedData,
	)
}

func UnpackSwapSimpleModeInputs(data []byte) (SwapSimpleModeInputs, error) {
	method, err := abis.MetaAggregationRouterV2Optimistic.MethodById(data)
	if err != nil {
		return SwapSimpleModeInputs{}, err
	}

	unpacked, err := method.Inputs.Unpack(data[4:])
	if err != nil {
		return SwapSimpleModeInputs{}, err
	}

	var inputs SwapDataCompress
	if err = method.Inputs.Copy(&inputs, unpacked); err != nil {
		return SwapSimpleModeInputs{}, err
	}

	return unpackSwapSimpleModeInputs(inputs.Data)
}

func packSwapSimpleModeInputs(swapInputs SwapSimpleModeInputs) ([]byte, error) {
	desc, err := packSwapDescriptionV2(swapInputs.Desc)
	if err != nil {
		return nil, err
	}

	return pack.Pack(
		swapInputs.Caller,
		pack.RawBytes(desc),
		pack.RawBytes(swapInputs.ExecutorData),
		swapInputs.ClientData,
	)
}

func unpackSwapSimpleModeInputs(data []byte) (SwapSimpleModeInputs, error) {
	var inputs SwapSimpleModeInputs
	var startByte int

	inputs.Caller, startByte = pack.ReadAddress(data, startByte)
	inputs.Desc, startByte = unpackSwapDescriptionV2(data, startByte)
	inputs.ExecutorData, startByte = unpackExecutorData(data, startByte)
	inputs.ClientData, _ = pack.ReadBytes(data, startByte)

	return inputs, nil
}

func unpackExecutorData(data []byte, startByte int) (ret []byte, endByte int) {
	endByte = startByte
	_, endByte = pack.ReadSliceAddress(data, endByte) // simpleSwapData.FirstPools
	_, endByte = pack.ReadSliceBigInt(data, endByte)  // simpleSwapData.FirstSwapAmounts
	_, endByte = pack.ReadSliceBytes(data, endByte)   // simpleSwapData.SwapDatas
	_, endByte = pack.ReadBigInt(data, endByte)       // simpleSwapData.Deadline
	_, endByte = pack.ReadBytes(data, endByte)        // simpleSwapData.DestTokenFeeData
	ret = data[startByte:endByte]
	return
}
