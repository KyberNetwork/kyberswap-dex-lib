package executor

import (
	"fmt"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common"
)

func packSwapSequencesNormalMode(
	chainID valueobject.ChainID,
	encodingRoute [][]types.EncodingSwap,
	executorAddress string,
	functionSelectorMappingID map[string]byte,
) ([]byte, error) {
	swapSequences := make([]pack.RawBytes, 0, len(encodingRoute))
	for _, encodingPath := range encodingRoute {
		swapDataPacked, err := packSwapData(chainID, encodingPath, executorAddress, functionSelectorMappingID)
		if err != nil {
			return nil, err
		}

		swapSequences = append(swapSequences, pack.RawBytes(swapDataPacked))
	}

	return pack.Pack(swapSequences)
}

func packSwapSequencesSimpleMode(
	chainID valueobject.ChainID,
	encodingRoute [][]types.EncodingSwap,
	executorAddress string,
	tokenIn string,
	functionSelectorMappingID map[string]byte,
) ([][]byte, error) {
	swapSequences := make([][]byte, 0, len(encodingRoute))
	for _, encodingPath := range encodingRoute {
		swapDataPacked, err := packSwapData(chainID, encodingPath, executorAddress, functionSelectorMappingID)
		if err != nil {
			return nil, err
		}
		swapDataPacked, err = pack.Pack(pack.RawBytes(swapDataPacked), common.HexToAddress(tokenIn))
		if err != nil {
			return nil, err
		}

		swapSequences = append(swapSequences, swapDataPacked)
	}

	return swapSequences, nil
}

func packSwapData(
	chainID valueobject.ChainID,
	encodingPath []types.EncodingSwap,
	executorAddress string,
	functionSelectorMappingID map[string]byte,
) ([]byte, error) {
	swapData := make([]pack.RawBytes, 0, len(encodingPath))

	for idx, encodingSwap := range encodingPath {
		poolMappingID := pack.UInt24(0) // poolMappingID always equal 0 for now
		var recipientFlag uint8 = 0
		if idx < len(encodingPath)-1 && encodingSwap.Recipient == encodingPath[idx+1].Pool {
			recipientFlag = 1
		} else if encodingSwap.Recipient == executorAddress {
			recipientFlag = 2
		}
		l2encodingSwap := types.L2EncodingSwap{
			EncodingSwap:  encodingSwap,
			PoolMappingID: poolMappingID,
			RecipientFlag: recipientFlag,
			IsFirstSwap:   idx == 0,
		}

		swapPacked, err := packSwap(
			chainID,
			l2encodingSwap,
			functionSelectorMappingID,
		)
		if err != nil {
			return nil, err
		}

		swapData = append(swapData, pack.RawBytes(swapPacked))
	}

	return pack.Pack(swapData)
}

func packSwap(
	chainID valueobject.ChainID,
	encodingSwap types.L2EncodingSwap,
	functionSelectorMappingID map[string]byte,
) ([]byte, error) {
	packSwapDataFunc, err := GetPackSwapDataFunc(encodingSwap.Exchange)
	if err != nil {
		return nil, err
	}

	data, err := packSwapDataFunc(chainID, encodingSwap)
	if err != nil {
		return nil, err
	}

	functionSelector, err := GetFunctionSelector(encodingSwap.Exchange)
	if err != nil {
		return nil, err
	}

	functionSelectorID, ok := functionSelectorMappingID[strings.ToLower(functionSelector.RawName)]
	if !ok {
		return nil, fmt.Errorf("no function selector id for %s", functionSelector.RawName)
	}

	return pack.Pack(data, functionSelectorID)
}

func unpackSwapSequencesNormalMode(data []byte, startByte int) (ret []byte, endByte int) {
	startByteSwapSequences := startByte
	swapSequencesLength, startByte := pack.ReadUInt8(data, startByte)
	for i := uint8(0); i < swapSequencesLength; i++ {
		var swapDataLength uint8
		swapDataLength, startByte = pack.ReadUInt8(data, startByte)
		for j := uint8(0); j < swapDataLength; j++ {
			_, startByte = pack.ReadBytes(data, startByte) // swap.Data
			_, startByte = pack.ReadUInt8(data, startByte) // swap.FunctionSelector
		}
	}
	endByte = startByte
	ret = data[startByteSwapSequences:endByte]
	return
}
