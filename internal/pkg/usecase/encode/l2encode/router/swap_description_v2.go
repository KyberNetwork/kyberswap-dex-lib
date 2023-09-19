package router

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/helper"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/ethereum/go-ethereum/common"
)

func buildSwapDescriptionV2ForSwap(executorAddress string, data types.EncodingData) SwapDescriptionV2 {
	minAmountOut := business.GetMinAmountOutExactInput(data.OutputAmount, data.SlippageTolerance)
	srcReceivers, srcAmounts := helper.GetSrcReceiversAndAmounts(data.TokenIn, data.Route, executorAddress)
	feeReceivers, feeAmounts := helper.GetFeeReceiversAndAmounts(data.ExtraFee)
	flagValues := helper.GetFlagsValue(data.Flags)

	swapDescription := SwapDescriptionV2{
		SrcToken:        common.HexToAddress(data.TokenIn),
		DstToken:        common.HexToAddress(data.TokenOut),
		SrcReceivers:    srcReceivers,
		SrcAmounts:      srcAmounts,
		FeeReceivers:    feeReceivers,
		FeeAmounts:      feeAmounts,
		DstReceiver:     common.HexToAddress(data.Recipient),
		Amount:          data.InputAmount,
		MinReturnAmount: minAmountOut,
		Flags:           uint32(flagValues),
		Permit:          data.Permit,
	}

	return swapDescription
}

func buildSwapDescriptionV2ForSwapSimpleMode(data types.EncodingData) SwapDescriptionV2 {
	minAmountOut := business.GetMinAmountOutExactInput(data.OutputAmount, data.SlippageTolerance)
	feeReceivers, feeAmounts := helper.GetFeeReceiversAndAmounts(data.ExtraFee)
	flagValues := helper.GetFlagsValue(data.Flags)

	swapDescription := SwapDescriptionV2{
		SrcToken:        common.HexToAddress(data.TokenIn),
		DstToken:        common.HexToAddress(data.TokenOut),
		SrcReceivers:    nil,
		SrcAmounts:      nil,
		FeeReceivers:    feeReceivers,
		FeeAmounts:      feeAmounts,
		DstReceiver:     common.HexToAddress(data.Recipient),
		Amount:          data.InputAmount,
		MinReturnAmount: minAmountOut,
		Flags:           uint32(flagValues),
		Permit:          data.Permit,
	}

	return swapDescription
}

func packSwapDescriptionV2(swapDescription SwapDescriptionV2) ([]byte, error) {
	return pack.Pack(
		swapDescription.SrcToken,
		swapDescription.DstToken,
		swapDescription.SrcReceivers,
		swapDescription.SrcAmounts,
		swapDescription.FeeReceivers,
		swapDescription.FeeAmounts,
		swapDescription.DstReceiver,
		swapDescription.Amount,
		swapDescription.MinReturnAmount,
		swapDescription.Flags,
		swapDescription.Permit,
	)
}

func unpackSwapDescriptionV2(data []byte, startByte int) (SwapDescriptionV2, int) {
	var swap SwapDescriptionV2
	swap.SrcToken, startByte = pack.ReadAddress(data, startByte)
	swap.DstToken, startByte = pack.ReadAddress(data, startByte)
	swap.SrcReceivers, startByte = pack.ReadSliceAddress(data, startByte)
	swap.SrcAmounts, startByte = pack.ReadSliceBigInt(data, startByte)
	swap.FeeReceivers, startByte = pack.ReadSliceAddress(data, startByte)
	swap.FeeAmounts, startByte = pack.ReadSliceBigInt(data, startByte)
	swap.DstReceiver, startByte = pack.ReadAddress(data, startByte)
	swap.Amount, startByte = pack.ReadBigInt(data, startByte)
	swap.MinReturnAmount, startByte = pack.ReadBigInt(data, startByte)
	swap.Flags, startByte = pack.ReadUInt32(data, startByte)
	swap.Permit, startByte = pack.ReadBytes(data, startByte)

	return swap, startByte
}
