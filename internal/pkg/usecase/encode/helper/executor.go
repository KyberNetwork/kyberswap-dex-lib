package helper

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/ethereum/go-ethereum/common"
)

const defaultMinimumPSThreshold = 1000000

// GetMinPositiveSlippageAmount returns an amount threshold,
// that we only take PS if the PS is higher than this threshold.
func GetMinPositiveSlippageAmount(outputAmount *big.Int, minimumPSThreshold int64) *big.Int {
	if minimumPSThreshold == 0 {
		minimumPSThreshold = defaultMinimumPSThreshold
	}

	minPSAmount := new(big.Int).Div(outputAmount, big.NewInt(minimumPSThreshold))
	if minPSAmount.Cmp(constant.One) == -1 {
		return constant.One
	}
	return minPSAmount
}

// ExtractFirstSwap returns the first addresses and first swap amounts
// for each path in a route. Useful when encoding simple swap data.
func ExtractFirstSwap(route [][]types.EncodingSwap) ([]common.Address, []*big.Int) {
	firstPools := make([]common.Address, 0, len(route))
	firstSwapAmounts := make([]*big.Int, 0, len(route))

	for _, path := range route {
		if len(path) == 0 {
			continue
		}

		firstPools = append(firstPools, common.HexToAddress(path[0].Pool))
		firstSwapAmounts = append(firstSwapAmounts, path[0].SwapAmount)
	}

	return firstPools, firstSwapAmounts
}
