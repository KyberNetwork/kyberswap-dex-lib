package executor

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

func PackPositiveSlippageFeeData(inputs PositiveSlippageFeeData) ([]byte, error) {
	// expectedReturnAmount: [minimumPSAmount (128 bits) + expectedReturnAmount (128 bits)]
	expectedReturnAmountPacked := new(big.Int).Set(inputs.MinimumPSAmount)
	expectedReturnAmountPacked.Lsh(expectedReturnAmountPacked, 128)
	expectedReturnAmountPacked.Or(expectedReturnAmountPacked, inputs.ExpectedReturnAmount)

	return PositiveSlippageFeeDataABIArguments.Pack(expectedReturnAmountPacked)
}

func UnpackPositiveSlippageFeeData(bytes []byte) (PositiveSlippageFeeData, error) {
	unpacked, err := PositiveSlippageFeeDataABIArguments.Unpack(bytes)
	if err != nil {
		return PositiveSlippageFeeData{}, err
	}

	var data PositiveSlippageFeeData
	if err = SwapSingleSequenceInputsABIArguments.Copy(&data, unpacked); err != nil {
		return PositiveSlippageFeeData{}, nil
	}

	bitmask := new(big.Int).Sub(new(big.Int).Lsh(constant.One, 128), constant.One)
	data.MinimumPSAmount = new(big.Int).Rsh(data.ExpectedReturnAmount, 128)
	data.ExpectedReturnAmount = data.ExpectedReturnAmount.And(data.ExpectedReturnAmount, bitmask)

	return data, nil
}

const defaultMinimumPSThreshold = 1000000

func getMinPositiveSlippageAmount(outputAmount *big.Int, minimumPSThreshold int64) *big.Int {
	if minimumPSThreshold == 0 {
		minimumPSThreshold = defaultMinimumPSThreshold
	}

	minPSAmount := new(big.Int).Div(outputAmount, big.NewInt(minimumPSThreshold))
	if minPSAmount.Cmp(constant.One) == -1 {
		return constant.One
	}
	return minPSAmount
}
