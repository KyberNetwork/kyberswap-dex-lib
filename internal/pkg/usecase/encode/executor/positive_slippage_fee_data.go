package executor

func PackPositiveSlippageFeeData(inputs PositiveSlippageFeeData) ([]byte, error) {
	return PositiveSlippageFeeDataABIArguments.Pack(inputs.ExpectedReturnAmount)
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

	return data, nil
}
