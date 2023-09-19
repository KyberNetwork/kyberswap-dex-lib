package router

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/business"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/helper"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
)

// buildSwapDescriptionV2ForSwap return SwapDescriptionV2 which is used by aggregation router in swap normal mode
// the main different between buildSwapDescriptionV2ForSwap and buildSwapDescriptionV2ForSwapSimpleMode is in normal mode,
// aggregation router need to transfer total amount of first swaps of each sequence to aggregation executor while
// in simple mode, those amounts is transferred to first pools directly
func buildSwapDescriptionV2ForSwap(executorAddress string, data types.EncodingData) SwapDescriptionV2 {
	minAmountOut := business.GetMinAmountOutExactInput(data.OutputAmount, data.SlippageTolerance)
	srcReceivers, srcAmounts := helper.GetSrcReceiversAndAmounts(data.TokenIn, data.Route, executorAddress)
	feeReceivers, feeAmounts := helper.GetFeeReceiversAndAmounts(data.ExtraFee)
	flagValues := helper.GetFlagsValue(data.Flags)

	return SwapDescriptionV2{
		SrcToken:        common.HexToAddress(data.TokenIn),
		DstToken:        common.HexToAddress(data.TokenOut),
		SrcReceivers:    srcReceivers,
		SrcAmounts:      srcAmounts,
		FeeReceivers:    feeReceivers,
		FeeAmounts:      feeAmounts,
		DstReceiver:     common.HexToAddress(data.Recipient),
		Amount:          data.InputAmount,
		MinReturnAmount: minAmountOut,
		Flags:           big.NewInt(flagValues),
		Permit:          data.Permit,
	}
}

// buildSwapDescriptionV2ForSwapSimpleMode return SwapDescriptionV2 which is used by aggregation router in swap simple mode
func buildSwapDescriptionV2ForSwapSimpleMode(data types.EncodingData) SwapDescriptionV2 {
	minAmountOut := business.GetMinAmountOutExactInput(data.OutputAmount, data.SlippageTolerance)
	feeReceivers, feeAmounts := helper.GetFeeReceiversAndAmounts(data.ExtraFee)
	flagValues := helper.GetFlagsValue(data.Flags)

	return SwapDescriptionV2{
		SrcToken:        common.HexToAddress(data.TokenIn),
		DstToken:        common.HexToAddress(data.TokenOut),
		SrcReceivers:    nil,
		SrcAmounts:      nil,
		FeeReceivers:    feeReceivers,
		FeeAmounts:      feeAmounts,
		DstReceiver:     common.HexToAddress(data.Recipient),
		Amount:          data.InputAmount,
		MinReturnAmount: minAmountOut,
		Flags:           big.NewInt(flagValues),
		Permit:          data.Permit,
	}
}
