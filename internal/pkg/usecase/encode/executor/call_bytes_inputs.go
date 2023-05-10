package executor

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func BuildAndPackCallBytesInputs(chainID valueobject.ChainID, routerAddress string, data types.EncodingData) ([]byte, error) {
	swapSequences, err := BuildSwapSequences(chainID, data.Route)
	if err != nil {
		return nil, err
	}

	minAmountOut := business.GetMinAmountOutExactInput(data.OutputAmount, data.SlippageTolerance)

	callBytesInputs := CallBytesInputs{
		Data: SwapExecutorDescription{
			SwapSequences:     swapSequences,
			TokenIn:           common.HexToAddress(data.TokenIn),
			TokenOut:          common.HexToAddress(data.TokenOut),
			MinTotalAmountOut: minAmountOut,
			To:                common.HexToAddress(routerAddress),
			Deadline:          data.Deadline,
			DestTokenFeeData:  nil,
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
