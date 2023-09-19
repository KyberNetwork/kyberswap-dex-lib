package decode

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/executor"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/router"
	l2executor "github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/executor"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/executor/swapdata"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/pack"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type L2Decoder struct {
	config Config
}

func NewL2Decoder(config Config) *L2Decoder {
	return &L2Decoder{config: config}
}

func (d *L2Decoder) Decode(data string) (interface{}, error) {
	decoded, err := hexutil.Decode(data)
	if err != nil {
		return nil, err
	}

	// We are using L1 router encoder instead of L2 router encoder,
	// since it yields a better optimization.
	method, err := abis.MetaAggregationRouterV2.MethodById(decoded)
	if err != nil {
		return nil, err
	}

	switch method.Name {
	case router.MethodNameSwap:
		return d.decodeSwapInputs(decoded)
	case router.MethodNameSwapSimpleMode:
		return d.decodeSwapSimpleMode(decoded)
	default:
		return nil, fmt.Errorf("unsupported method: [%s]", method.Name)
	}
}

func (d *L2Decoder) decodeSwapInputs(data []byte) (interface{}, error) {
	swapInputs, err := router.UnpackSwapInputs(data)
	if err != nil {
		return nil, err
	}

	targetData, err := d.decodeCallBytesInputs(swapInputs.Execution.TargetData)
	if err != nil {
		return nil, err
	}

	var clientData DecodedClientData
	if err = json.Unmarshal(swapInputs.Execution.ClientData, &clientData); err != nil {
		return nil, err
	}

	return DecodedSwapInputs{
		Execution: DecodedSwapExecutionParams{
			CallTarget:    swapInputs.Execution.CallTarget,
			ApproveTarget: swapInputs.Execution.ApproveTarget,
			TargetData:    targetData,
			Desc:          swapInputs.Execution.Desc,
			ClientData:    clientData,
		},
	}, nil
}

func (d *L2Decoder) decodeSwapSimpleMode(data []byte) (interface{}, error) {
	swapSimpleModeInputs, err := router.UnpackSwapSimpleModeInputs(data)
	if err != nil {
		return DecodedSwapSimpleModeInputs{}, err
	}

	executorData, err := d.decodeSimpleSwapData(swapSimpleModeInputs.ExecutorData)
	if err != nil {
		return DecodedSwapSimpleModeInputs{}, err
	}

	return DecodedSwapSimpleModeInputs{
		Caller:       swapSimpleModeInputs.Caller,
		Desc:         swapSimpleModeInputs.Desc,
		ExecutorData: executorData,
	}, nil
}

func (d *L2Decoder) decodeCallBytesInputs(data []byte) (DecodedCallBytesInputs, error) {
	callBytesInputs, err := l2executor.UnpackCallBytesInputs(data)
	if err != nil {
		return DecodedCallBytesInputs{}, nil
	}

	decodedSwapSequences := d.decodeSwapSequencesNormalMode(callBytesInputs.SwapSequences)

	var positiveSlippageFeeData executor.PositiveSlippageFeeData
	if len(callBytesInputs.PositiveSlippageData) > 0 {
		positiveSlippageFeeData, err = executor.UnpackPositiveSlippageFeeData(callBytesInputs.PositiveSlippageData)
		if err != nil {
			return DecodedCallBytesInputs{}, nil
		}
	}

	return DecodedCallBytesInputs{
		Data: DecodedSwapExecutorDescription{
			SwapSequences:    decodedSwapSequences,
			TokenIn:          callBytesInputs.TokenIn,
			TokenOut:         callBytesInputs.TokenOut,
			To:               callBytesInputs.To,
			Deadline:         callBytesInputs.Deadline,
			DestTokenFeeData: positiveSlippageFeeData,
		},
	}, nil
}

func (d *L2Decoder) decodeSimpleSwapData(data []byte) (DecodedSimpleSwapData, error) {
	simpleSwapData, err := l2executor.UnpackSimpleSwapData(data)
	if err != nil {
		return DecodedSimpleSwapData{}, err
	}

	decodedSwapSequences := d.decodeSwapSequencesSimpleMode(simpleSwapData.SwapDatas)

	var positiveSlippageFeeData executor.PositiveSlippageFeeData
	if len(simpleSwapData.DestTokenFeeData) > 0 {
		positiveSlippageFeeData, err = executor.UnpackPositiveSlippageFeeData(simpleSwapData.DestTokenFeeData)
		if err != nil {
			return DecodedSimpleSwapData{}, nil
		}
	}

	return DecodedSimpleSwapData{
		FirstPools:       simpleSwapData.FirstPools,
		FirstSwapAmounts: simpleSwapData.FirstSwapAmounts,
		SwapDatas:        decodedSwapSequences,
		Deadline:         simpleSwapData.Deadline,
		DestTokenFeeData: positiveSlippageFeeData,
	}, nil
}

func (d *L2Decoder) decodeSwapSequencesNormalMode(data []byte) [][]DecodedSwap {
	var startByte int
	swapSequencesLength, startByte := pack.ReadUInt8(data, startByte)

	swapSequences := make([][]DecodedSwap, swapSequencesLength)
	for i := uint8(0); i < swapSequencesLength; i++ {
		swapSequences[i], startByte = d.decodeSwapPath(data, startByte)
	}

	return swapSequences
}

func (d *L2Decoder) decodeSwapSequencesSimpleMode(data [][]byte) [][]DecodedSwap {
	swapSequences := make([][]DecodedSwap, len(data))
	for i, swapDataByte := range data {
		swapSequences[i], _ = d.decodeSwapPath(swapDataByte, 0)
	}

	return swapSequences
}

func (d *L2Decoder) decodeSwapPath(data []byte, startByte int) ([]DecodedSwap, int) {
	swapPathLength, startByte := pack.ReadUInt8(data, startByte)
	swapPath := make([]DecodedSwap, swapPathLength)

	for i := uint8(0); i < swapPathLength; i++ {
		swapPath[i], startByte = d.decodeSwap(data, startByte, i == 0)
	}
	return swapPath, startByte
}

func (d *L2Decoder) decodeSwap(data []byte, startByte int, isFirstSwap bool) (DecodedSwap, int) {
	// Read function selector
	swapDataBytes, startByte := pack.ReadBytes(data, startByte)
	functionSelectorId, startByte := pack.ReadUInt8(data, startByte)
	var functionSelector string
	for k, v := range d.config.FunctionSelectorMappingID {
		if v == functionSelectorId {
			functionSelector = k
			break
		}
	}

	swapData, err := d.decodeSwapData(swapDataBytes, functionSelector, isFirstSwap)
	if err != nil {
		// Fallback to swapDataBytes
		swapData = common.Bytes2Hex(swapDataBytes)
	}

	return DecodedSwap{
		Data:             swapData,
		FunctionSelector: functionSelector,
	}, startByte
}

func (d *L2Decoder) decodeSwapData(data []byte, functionSelectorName string, isFirstSwap bool) (interface{}, error) {
	switch functionSelectorName {
	case strings.ToLower(l2executor.FunctionSelectorBalancerV2.RawName):
		return swapdata.UnpackBalancerV2(data, isFirstSwap)
	case strings.ToLower(l2executor.FunctionSelectorCamelotSwap.RawName):
		return swapdata.UnpackCamelot(data, isFirstSwap)
	case strings.ToLower(l2executor.FunctionSelectorCurveSwap.RawName):
		return swapdata.UnpackCurveSwap(data, isFirstSwap)
	case strings.ToLower(l2executor.FunctionSelectorDODO.RawName):
		return swapdata.UnpackDODO(data, isFirstSwap)
	case strings.ToLower(l2executor.FunctionSelectorFraxSwap.RawName):
		return swapdata.UnpackFraxSwap(data, isFirstSwap)
	case strings.ToLower(l2executor.FunctionSelectorGMX.RawName):
		return swapdata.UnpackGMX(data, isFirstSwap)
	case strings.ToLower(l2executor.FunctionSelectorLimitOrder.RawName):
		return swapdata.UnpackKyberLimitOrder(data, isFirstSwap)
	case strings.ToLower(l2executor.FunctionSelectorStableSwap.RawName):
		return swapdata.UnpackStableSwap(data, isFirstSwap)
	case strings.ToLower(l2executor.FunctionSelectorSynthetix.RawName):
		return swapdata.UnpackSynthetix(data, isFirstSwap)
	case strings.ToLower(l2executor.FunctionSelectorUniV3KSElastic.RawName):
		return swapdata.UnpackUniswapV3KSElastic(data, isFirstSwap)
	case strings.ToLower(l2executor.FunctionSelectorKSClassic.RawName),
		strings.ToLower(l2executor.FunctionSelectorUniswap.RawName),
		strings.ToLower(l2executor.FunctionSelectorVelodrome.RawName):
		return swapdata.UnpackUniswap(data, isFirstSwap)

	default:
		return nil, fmt.Errorf("unsupported function selector")
	}
}
