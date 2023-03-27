package executor

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase/types"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

func BuildAndPackSwapSequences(chainID valueobject.ChainID, encodingRoute [][]types.EncodingSwap) ([][]byte, error) {
	swapSequences := make([][]byte, 0, len(encodingRoute))

	for _, encodingPath := range encodingRoute {
		swapData := make([]Swap, 0, len(encodingPath))

		for _, encodingSwap := range encodingPath {
			swp, err := BuildSwap(chainID, encodingSwap)
			if err != nil {
				return nil, err
			}

			swapData = append(swapData, swp)
		}

		swapSingleSequenceInputs := SwapSingleSequenceInputs{
			SwapData: swapData,
		}

		packedSwapSimpleModeInputs, err := PackSwapSingleSequenceInputs(swapSingleSequenceInputs)
		if err != nil {
			return nil, err
		}

		swapSequences = append(swapSequences, packedSwapSimpleModeInputs)
	}

	return swapSequences, nil
}

func BuildSwapSequences(chainID valueobject.ChainID, encodingRoute [][]types.EncodingSwap) ([][]Swap, error) {
	swapSequences := make([][]Swap, 0, len(encodingRoute))

	for _, encodingPath := range encodingRoute {
		swapSequence := make([]Swap, 0, len(encodingPath))

		for _, encodingSwap := range encodingPath {
			swap, err := BuildSwap(chainID, encodingSwap)
			if err != nil {
				return nil, err
			}

			swapSequence = append(swapSequence, swap)
		}

		swapSequences = append(swapSequences, swapSequence)
	}

	return swapSequences, nil
}

func BuildSwap(chainID valueobject.ChainID, encodingSwap types.EncodingSwap) (Swap, error) {
	packSwapDataFunc, err := GetPackSwapDataFunc(encodingSwap.Exchange)
	if err != nil {
		return Swap{}, err
	}

	data, err := packSwapDataFunc(chainID, encodingSwap)
	if err != nil {
		return Swap{}, err
	}

	functionSelector, err := GetFunctionSelector(encodingSwap.Exchange)
	if err != nil {
		return Swap{}, err
	}

	return Swap{
		Data:             data,
		FunctionSelector: functionSelector.ID,
	}, nil
}

func PackSwapSingleSequenceInputs(inputs SwapSingleSequenceInputs) ([]byte, error) {
	return SwapSingleSequenceInputsABIArguments.Pack(inputs.SwapData)
}

func UnpackSwapSingleSequenceInputs(data []byte) (SwapSingleSequenceInputs, error) {
	unpacked, err := SwapSingleSequenceInputsABIArguments.Unpack(data)
	if err != nil {
		return SwapSingleSequenceInputs{}, err
	}

	var inputs SwapSingleSequenceInputs
	if err = SwapSingleSequenceInputsABIArguments.Copy(&inputs, unpacked); err != nil {
		return SwapSingleSequenceInputs{}, nil
	}

	return inputs, nil
}
