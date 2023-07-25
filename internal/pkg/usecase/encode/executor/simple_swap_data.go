package executor

import (
	"encoding/hex"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

var (
	// OffsetToTheStartOfData 32 bytes string
	// https://docs.soliditylang.org/en/develop/abi-spec.html#use-of-dynamic-types
	OffsetToTheStartOfData = "0000000000000000000000000000000000000000000000000000000000000020"
)

func BuildAndPackSimpleSwapData(chainID valueobject.ChainID, _ string, isPositiveSlippageEnabled bool, minimumPSThreshold int64, data types.EncodingData) ([]byte, error) {
	swapDatas, err := BuildAndPackSwapSequences(chainID, data.Route)
	if err != nil {
		return nil, err
	}

	firstPools, firstSwapAmounts := extractFirstSwap(data.Route)

	var destTokenFeeData []byte
	if isPositiveSlippageEnabled {
		positiveSlippageFeeData := PositiveSlippageFeeData{
			ExpectedReturnAmount: data.TotalAmountOut,
			MinimumPSAmount:      getMinPositiveSlippageAmount(data.TotalAmountOut, minimumPSThreshold),
		}

		destTokenFeeData, err = PackPositiveSlippageFeeData(positiveSlippageFeeData)
		if err != nil {
			return nil, err
		}
	}

	simpleSwapData := SimpleSwapData{
		FirstPools:       firstPools,
		FirstSwapAmounts: firstSwapAmounts,
		SwapDatas:        swapDatas,
		Deadline:         data.Deadline,
		DestTokenFeeData: destTokenFeeData,
	}

	return PackSimpleSwapData(simpleSwapData)
}

func PackSimpleSwapData(data SimpleSwapData) ([]byte, error) {
	packedSimpleSwapData, err := SimpleSwapDataABIArguments.Pack(
		data.FirstPools,
		data.FirstSwapAmounts,
		data.SwapDatas,
		data.Deadline,
		data.DestTokenFeeData,
	)
	if err != nil {
		return nil, err
	}

	return hex.DecodeString(OffsetToTheStartOfData + common.Bytes2Hex(packedSimpleSwapData))
}

func UnpackSimpleSwapData(data []byte) (SimpleSwapData, error) {
	encodeString := hex.EncodeToString(data)

	packedSimpleSwapDataHex := strings.Replace(encodeString, OffsetToTheStartOfData, "", 1)

	packedSimpleSwapData := common.Hex2Bytes(packedSimpleSwapDataHex)

	unpacked, err := SimpleSwapDataABIArguments.Unpack(packedSimpleSwapData)
	if err != nil {
		return SimpleSwapData{}, err
	}

	var simpleSwapData SimpleSwapData
	if err = SimpleSwapDataABIArguments.Copy(&simpleSwapData, unpacked); err != nil {
		return SimpleSwapData{}, err
	}

	return simpleSwapData, nil
}

func extractFirstSwap(route [][]types.EncodingSwap) ([]common.Address, []*big.Int) {
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
