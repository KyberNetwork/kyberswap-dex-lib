package executor

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/helper"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/executor"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// PackSimpleSwapData only uses L2 Optimize for swapSequences.
// Other field will be pack as default L1 encode logic.
func PackSimpleSwapData(
	chainID valueobject.ChainID,
	routerAddress, executorAddress string,
	functionSelectorMappingID map[string]byte,
	isPositiveSlippageEnabled bool,
	minimumPSThreshold int64,
	data types.EncodingData,
) ([]byte, error) {
	swapSequences, err := packSwapSequencesSimpleMode(chainID, data.Route, executorAddress, data.TokenIn, functionSelectorMappingID)
	if err != nil {
		return nil, err
	}

	firstPools, firstSwapAmounts := helper.ExtractFirstSwap(data.Route)

	var positiveSlippageFeeDataPacked []byte
	if isPositiveSlippageEnabled {
		positiveSlippageFeeData := executor.PositiveSlippageFeeData{
			ExpectedReturnAmount: data.TotalAmountOut,
			MinimumPSAmount:      helper.GetMinPositiveSlippageAmount(data.TotalAmountOut, minimumPSThreshold),
		}

		positiveSlippageFeeDataPacked, err = executor.PackPositiveSlippageFeeData(positiveSlippageFeeData)
		if err != nil {
			return nil, err
		}
	}

	simpleSwapData := executor.SimpleSwapData{
		FirstPools:       firstPools,
		FirstSwapAmounts: firstSwapAmounts,
		SwapDatas:        swapSequences,
		Deadline:         data.Deadline,
		DestTokenFeeData: positiveSlippageFeeDataPacked,
	}

	return executor.PackSimpleSwapData(simpleSwapData)
}

func UnpackSimpleSwapData(data []byte) (executor.SimpleSwapData, error) {
	return executor.UnpackSimpleSwapData(data)
}
