package executor

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/stretchr/testify/assert"
)

func TestPackingSwapSequences(t *testing.T) {
	t.Parallel()

	swapAmount, _ := new(big.Int).SetString("1000000000000000000", 10)
	amountOut, _ := new(big.Int).SetString("1869546121", 10)

	testCases := []struct {
		name            string
		chainID         valueobject.ChainID
		encodingRoute   [][]types.EncodingSwap
		packFunc        func(chainID valueobject.ChainID, encodingRoute [][]types.EncodingSwap, executorAddress string, functionSelectorMappingID map[string]byte) ([]byte, error)
		executorAddress string
		expectedResult  string
	}{
		{
			name:    "it should pack swap sequences normal mode correctly",
			chainID: valueobject.ChainIDOptimism,
			encodingRoute: [][]types.EncodingSwap{
				{
					{
						Pool:              "0x683860e93fab18b8e2e52f9be310d9b36b49677a",
						TokenIn:           "0x4200000000000000000000000000000000000006",
						TokenOut:          "0x7f5c764cbc14f9669b88837ca1490cca17c31607",
						SwapAmount:        swapAmount,
						AmountOut:         amountOut,
						LimitReturnAmount: big.NewInt(0),
						Exchange:          valueobject.ExchangeKyberswapElastic,
						PoolLength:        2,
						PoolType:          "elastic",
						PoolExtra:         nil,
						Extra:             "{}",
						Flags:             []types.EncodingSwapFlag{{Value: 0x02}},
						CollectAmount:     swapAmount,
						Recipient:         "0xcaa00aaf6fbc769d627d825b4faedc3aad880597",
					},
				},
			},
			packFunc:        packSwapSequencesNormalMode,
			executorAddress: "0xcaa00aaf6fbc769d627d825b4faedc3aad880597",
			expectedResult:  "01010000002902000000683860e93fab18b8e2e52f9be310d9b36b49677a00000000000000000de0b6b3a76400000008",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := tc.packFunc(tc.chainID, tc.encodingRoute, tc.executorAddress, map[string]byte{"executeuniv3kselastic": 8})
			assert.ErrorIs(t, err, nil)
			assert.Equal(t, tc.expectedResult, hex.EncodeToString(result))
		})
	}
}
