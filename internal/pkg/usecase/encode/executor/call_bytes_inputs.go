package executor

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func BuildAndPackCallBytesInputs(chainID valueobject.ChainID, routerAddress string, isPositiveSlippageEnabled bool, data types.EncodingData) ([]byte, error) {
	swapSequences, err := BuildSwapSequences(chainID, data.Route)
	if err != nil {
		return nil, err
	}

	var destTokenFeeData []byte
	if isPositiveSlippageEnabled {
		positiveSlippageFeeData := PositiveSlippageFeeData{
			ExpectedReturnAmount: data.TotalAmountOut,
		}

		destTokenFeeData, err = PackPositiveSlippageFeeData(positiveSlippageFeeData)
		if err != nil {
			return nil, err
		}
	}

	callBytesInputs := CallBytesInputs{
		Data: SwapExecutorDescription{
			SwapSequences:    swapSequences,
			TokenIn:          common.HexToAddress(data.TokenIn),
			TokenOut:         common.HexToAddress(data.TokenOut),
			To:               common.HexToAddress(routerAddress),
			Deadline:         data.Deadline,
			DestTokenFeeData: destTokenFeeData,
		},
	}

	return PackCallBytesInputs(callBytesInputs)
}

func PackCallBytesInputs(callBytesInputs CallBytesInputs) ([]byte, error) {
	return CallBytesInputsABIArguments.Pack(callBytesInputs.Data)
}

func UnpackCallBytesInputs(data []byte) (CallBytesInputs, error) {
	unpacked, err := CallBytesInputsABIArguments.Unpack(data)
	if err != nil {
		return CallBytesInputs{}, err
	}

	var inputs CallBytesInputs
	if err = CallBytesInputsABIArguments.Copy(&inputs, unpacked); err != nil {
		return CallBytesInputs{}, nil
	}

	return inputs, nil
}
