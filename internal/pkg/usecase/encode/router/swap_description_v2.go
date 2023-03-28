package router

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/core"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// buildSwapDescriptionV2ForSwap return SwapDescriptionV2 which is used by aggregation router in swap normal mode
// the main different between buildSwapDescriptionV2ForSwap and buildSwapDescriptionV2ForSwapSimpleMode is in normal mode,
// aggregation router need to transfer total amount of first swaps of each sequence to aggregation executor while
// in simple mode, those amounts is transferred to first pools directly
func buildSwapDescriptionV2ForSwap(executorAddress string, data types.EncodingData) SwapDescriptionV2 {
	minAmountOut := core.GetMinAmountOutExactInput(data.OutputAmount, data.SlippageTolerance)
	srcReceivers, srcAmounts := getSrcReceiversAndAmounts(data.TokenIn, data.Route, executorAddress)
	feeReceivers, feeAmounts := getFeeReceiversAndAmounts(data.ExtraFee)
	flagValues := getFlagsValue(data.Flags)

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
	minAmountOut := core.GetMinAmountOutExactInput(data.OutputAmount, data.SlippageTolerance)
	feeReceivers, feeAmounts := getFeeReceiversAndAmounts(data.ExtraFee)
	flagValues := getFlagsValue(data.Flags)

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

// getSrcReceiversAndAmounts returns a list of address and a list of amount which aggregation router should transfer token to
// In case swap (normal mode), if tokenIn is no ether (no need to unwrap), aggregation router should transfer to aggregation executor
// total amount of first swaps of each swap sequence (path)
func getSrcReceiversAndAmounts(tokenIn string, route [][]types.EncodingSwap, executorAddress string) ([]common.Address, []*big.Int) {
	receivers := make([]common.Address, 0, 1)
	amounts := make([]*big.Int, 0, 1)

	if !eth.IsEther(tokenIn) {
		receivers = append(receivers, common.HexToAddress(executorAddress))
		amounts = append(amounts, getFirstSwapAmount(route))
	}

	return receivers, amounts
}

// getFeeReceiversAndAmounts returns a list of address and a list of amount which aggregation router should transfer extra fee to
func getFeeReceiversAndAmounts(extraFee valueobject.ExtraFee) ([]common.Address, []*big.Int) {
	receivers := make([]common.Address, 0, 1)
	amounts := make([]*big.Int, 0, 1)

	if extraFee.FeeAmount != nil && len(extraFee.FeeReceiver) > 0 {
		receivers = append(receivers, common.HexToAddress(extraFee.FeeReceiver))
		amounts = append(amounts, extraFee.FeeAmount)
	}

	return receivers, amounts
}

// getFlagsValue returns value of flags
func getFlagsValue(flags []types.EncodingFlag) int64 {
	var flagsValue int64

	for _, flag := range flags {
		flagsValue = flagsValue | flag.Value
	}

	return flagsValue
}

// getFirstSwapAmount returns total amount of first swaps of each swap sequence (path)
func getFirstSwapAmount(route [][]types.EncodingSwap) *big.Int {
	firstSwapAmount := big.NewInt(0)

	for _, path := range route {
		if len(path) == 0 {
			continue
		}

		firstSwapAmount = new(big.Int).Add(firstSwapAmount, path[0].SwapAmount)
	}

	return firstSwapAmount
}
