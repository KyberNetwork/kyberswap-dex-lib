package executor

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/helper"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
)

func PackCallBytesInputs(
	chainID valueobject.ChainID,
	routerAddress, executorAddress string,
	functionSelectorMappingID map[string]byte,
	isPositiveSlippageEnabled bool,
	minimumPSThreshold int64,
	data types.EncodingData,
) ([]byte, error) {
	swapSequences, err := packSwapSequencesNormalMode(chainID, data.Route, executorAddress, functionSelectorMappingID)
	if err != nil {
		return nil, err
	}

	var positiveSlippageFeeDataPacked []byte
	if isPositiveSlippageEnabled {
		positiveSlippageFeeData := PositiveSlippageFeeData{
			PartnerReceiver:      pack.UInt160(big.NewInt(0)),
			PartnerPercent:       pack.UInt96(big.NewInt(0)),
			ExpectedReturnAmount: data.TotalAmountOut,
			MinimumPSAmount:      helper.GetMinPositiveSlippageAmount(data.TotalAmountOut, minimumPSThreshold),
		}

		positiveSlippageFeeDataPacked, err = PackPositiveSlippageFeeData(positiveSlippageFeeData)
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

	desc := SwapExecutorDescription{
		SwapSequences:        swapSequences,
		TokenIn:              common.HexToAddress(data.TokenIn),
		TokenOut:             common.HexToAddress(data.TokenOut),
		To:                   to,
		Deadline:             data.Deadline,
		PositiveSlippageData: positiveSlippageFeeDataPacked,
	}
	return packCallBytesInputs(desc)
}

func UnpackCallBytesInputs(data []byte) (SwapExecutorDescription, error) {
	var desc SwapExecutorDescription
	var startByte int

	desc.SwapSequences, startByte = unpackSwapSequencesNormalMode(data, startByte)
	desc.TokenIn, startByte = pack.ReadAddress(data, startByte)
	desc.TokenOut, startByte = pack.ReadAddress(data, startByte)
	desc.To, startByte = pack.ReadAddress(data, startByte)
	desc.Deadline, startByte = pack.ReadBigInt(data, startByte)
	desc.PositiveSlippageData, _ = pack.ReadBytes(data, startByte)
	return desc, nil
}

func packCallBytesInputs(desc SwapExecutorDescription) ([]byte, error) {
	return pack.Pack(
		pack.RawBytes(desc.SwapSequences),
		desc.TokenIn,
		desc.TokenOut,
		desc.To,
		desc.Deadline,
		desc.PositiveSlippageData,
	)
}
