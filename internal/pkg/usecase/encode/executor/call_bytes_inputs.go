package executor

import (
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

func BuildAndPackCallBytesInputs(chainID valueobject.ChainID, routerAddress string, isPositiveSlippageEnabled bool, minimumPSThreshold int64, data types.EncodingData) ([]byte, error) {
	swapSequences, err := BuildSwapSequences(chainID, data.Route)
	if err != nil {
		return nil, err
	}

	var destTokenFeeData []byte
	if isPositiveSlippageEnabled {
		positiveSlippageFeeData := PositiveSlippageFeeData{
			ExpectedReturnAmount: data.TotalAmountOut,
			MinimumPSAmount:      getMinPositiveSlippageAmount(data.TotalAmountOut, minimumPSThreshold),
		}

		destTokenFeeData, err = PackPositiveSlippageFeeData(positiveSlippageFeeData)
		if err != nil {
			return nil, err
		}
	}

	var to common.Address
	if data.ExtraFee.IsChargeFeeByCurrencyOut() &&
		data.ExtraFee.FeeAmount != nil &&
		data.ExtraFee.FeeAmount.Cmp(constant.Zero) > 0 {
		to = common.HexToAddress(routerAddress)
	} else {
		to = common.HexToAddress(data.Recipient)
	}

	callBytesInputs := CallBytesInputs{
		Data: SwapExecutorDescription{
			SwapSequences:    swapSequences,
			TokenIn:          common.HexToAddress(data.TokenIn),
			TokenOut:         common.HexToAddress(data.TokenOut),
			To:               to,
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
