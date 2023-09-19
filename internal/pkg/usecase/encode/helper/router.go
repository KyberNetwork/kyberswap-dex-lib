// Package helper provides helpful functions that use in both L1 & L2 encode.
package helper

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/eth"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
)

// GetSrcReceiversAndAmounts returns a list of address and a list of amount which aggregation router should transfer token to
// In case swap (normal mode), if tokenIn is no ether (no need to unwrap), aggregation router should transfer to aggregation executor
// total amount of first swaps of each swap sequence (path)
func GetSrcReceiversAndAmounts(tokenIn string, route [][]types.EncodingSwap, executorAddress string) ([]common.Address, []*big.Int) {
	receivers := make([]common.Address, 0, 1)
	amounts := make([]*big.Int, 0, 1)

	if !eth.IsEther(tokenIn) {
		receivers = append(receivers, common.HexToAddress(executorAddress))
		amounts = append(amounts, getFirstSwapAmount(route))
	}

	return receivers, amounts
}

// GetFeeReceiversAndAmounts returns a list of address and a list of amount which aggregation router should transfer extra fee to
func GetFeeReceiversAndAmounts(extraFee valueobject.ExtraFee) ([]common.Address, []*big.Int) {
	receivers := make([]common.Address, 0, 1)
	amounts := make([]*big.Int, 0, 1)

	if extraFee.FeeAmount != nil && len(extraFee.FeeReceiver) > 0 {
		receivers = append(receivers, common.HexToAddress(extraFee.FeeReceiver))
		amounts = append(amounts, extraFee.FeeAmount)
	}

	return receivers, amounts
}

// GetFlagsValue returns value of flags
func GetFlagsValue(flags []types.EncodingFlag) int64 {
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
