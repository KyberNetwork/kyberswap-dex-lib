package decode

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/router-service/internal/pkg/abis"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/executor"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/executor/swapdata"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/router"
)

type (
	DecodedSwapInputs struct {
		Execution DecodedSwapExecutionParams `json:"execution"`
	}

	DecodedSwapSimpleModeInputs struct {
		Caller       common.Address           `json:"caller"`
		Desc         router.SwapDescriptionV2 `json:"desc"`
		ExecutorData DecodedSimpleSwapData    `json:"executorData"`
	}

	DecodedSwapExecutionParams struct {
		CallTarget    common.Address           `json:"callTarget"`
		ApproveTarget common.Address           `json:"approveTarget"`
		TargetData    DecodedCallBytesInputs   `json:"targetData"`
		Desc          router.SwapDescriptionV2 `json:"desc"`
		ClientData    DecodedClientData        `json:"clientData"`
	}

	DecodedClientData struct {
		Source       string
		AmountInUSD  string
		AmountOutUSD string
		Referral     string
		Flags        uint32
	}

	DecodedCallBytesInputs struct {
		Data DecodedSwapExecutorDescription `json:"data"`
	}

	DecodedSwapExecutorDescription struct {
		SwapSequences    [][]DecodedSwap                  `json:"swapSequences"`
		TokenIn          common.Address                   `json:"tokenIn"`
		TokenOut         common.Address                   `json:"tokenOut"`
		To               common.Address                   `json:"to"`
		Deadline         *big.Int                         `json:"deadline"`
		DestTokenFeeData executor.PositiveSlippageFeeData `json:"destTokenFeeData"`
	}

	DecodedSwap struct {
		Data             interface{}        `json:"data"`
		FunctionSelector string             `json:"functionSelector"`
		Flags            executor.SwapFlags `json:"flags"`
	}

	DecodedSimpleSwapData struct {
		FirstPools       []common.Address                 `json:"firstPools"`
		FirstSwapAmounts []*big.Int                       `json:"firstSwapAmounts"`
		SwapDatas        [][]DecodedSwap                  `json:"swapDatas"`
		Deadline         *big.Int                         `json:"deadline"`
		DestTokenFeeData executor.PositiveSlippageFeeData `json:"destTokenFeeData"`
	}
)

type Decoder struct{}

func (d *Decoder) Decode(data string) (interface{}, error) {
	decoded, err := hexutil.Decode(data)
	if err != nil {
		return nil, err
	}

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

func (d *Decoder) decodeSwapInputs(data []byte) (interface{}, error) {
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

func (d *Decoder) decodeSwapSimpleMode(data []byte) (interface{}, error) {
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

func (d *Decoder) decodeCallBytesInputs(data []byte) (DecodedCallBytesInputs, error) {
	callBytesInputs, err := executor.UnpackCallBytesInputs(data)
	if err != nil {
		return DecodedCallBytesInputs{}, nil
	}

	decodedSwapSequences, err := d.decodeSwapSequences(callBytesInputs.Data.SwapSequences)
	if err != nil {
		return DecodedCallBytesInputs{}, nil
	}

	var positiveSlippageFeeData executor.PositiveSlippageFeeData
	if len(callBytesInputs.Data.DestTokenFeeData) > 0 {
		positiveSlippageFeeData, err = executor.UnpackPositiveSlippageFeeData(callBytesInputs.Data.DestTokenFeeData)
		if err != nil {
			return DecodedCallBytesInputs{}, nil
		}
	}

	return DecodedCallBytesInputs{
		Data: DecodedSwapExecutorDescription{
			SwapSequences:    decodedSwapSequences,
			TokenIn:          callBytesInputs.Data.TokenIn,
			TokenOut:         callBytesInputs.Data.TokenOut,
			To:               callBytesInputs.Data.To,
			Deadline:         callBytesInputs.Data.Deadline,
			DestTokenFeeData: positiveSlippageFeeData,
		},
	}, nil
}

func (d *Decoder) decodeSimpleSwapData(data []byte) (DecodedSimpleSwapData, error) {
	simpleSwapData, err := executor.UnpackSimpleSwapData(data)
	if err != nil {
		return DecodedSimpleSwapData{}, err
	}

	swapDatas, err := d.decodeSwapDatas(simpleSwapData.SwapDatas)
	if err != nil {
		return DecodedSimpleSwapData{}, err
	}

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
		SwapDatas:        swapDatas,
		Deadline:         simpleSwapData.Deadline,
		DestTokenFeeData: positiveSlippageFeeData,
	}, nil
}

func (d *Decoder) decodeSwapDatas(swapDatas [][]byte) ([][]DecodedSwap, error) {
	decodedSwapSequences := make([][]DecodedSwap, 0, len(swapDatas))
	for _, swapSequence := range swapDatas {
		decodedSwapSequence := make([]DecodedSwap, 0, len(swapSequence))

		swapSingleSequenceInputs, err := executor.UnpackSwapSingleSequenceInputs(swapSequence)
		if err != nil {
			return nil, err
		}

		for _, swp := range swapSingleSequenceInputs.SwapData {
			decodedSwap, err := d.decodeSwap(swp)
			if err != nil {
				return nil, err
			}

			decodedSwapSequence = append(decodedSwapSequence, decodedSwap)
		}

		decodedSwapSequences = append(decodedSwapSequences, decodedSwapSequence)
	}

	return decodedSwapSequences, nil
}

func (d *Decoder) decodeSwapSequences(swapSequences [][]executor.Swap) ([][]DecodedSwap, error) {
	decodedSwapSequences := make([][]DecodedSwap, 0, len(swapSequences))
	for _, swapSequence := range swapSequences {
		decodedSwapSequence := make([]DecodedSwap, 0, len(swapSequence))
		for _, swp := range swapSequence {
			decodedSwap, err := d.decodeSwap(swp)
			if err != nil {
				return nil, err
			}

			decodedSwapSequence = append(decodedSwapSequence, decodedSwap)
		}
		decodedSwapSequences = append(decodedSwapSequences, decodedSwapSequence)
	}

	return decodedSwapSequences, nil
}

func (d *Decoder) decodeSwap(swap executor.Swap) (DecodedSwap, error) {
	swapData, err := d.decodeSwapData(swap)
	if err != nil {
		return DecodedSwap{}, err
	}

	functionSelector, flags := d.decodeSelectorAndFlags(swap.SelectorAndFlags)

	return DecodedSwap{
		Data:             swapData,
		FunctionSelector: d.decodeFunctionSelector(functionSelector),
		Flags:            flags,
	}, nil
}

func (d *Decoder) decodeFunctionSelector(id executor.SwapSelector) string {
	switch id {
	case executor.FunctionSelectorUniswap.ID:
		return executor.FunctionSelectorUniswap.RawName
	case executor.FunctionSelectorStableSwap.ID:
		return executor.FunctionSelectorStableSwap.RawName
	case executor.FunctionSelectorCurveSwap.ID:
		return executor.FunctionSelectorCurveSwap.RawName
	case executor.FunctionSelectorUniV3KSElastic.ID:
		return executor.FunctionSelectorUniV3KSElastic.RawName
	case executor.FunctionSelectorBalancerV2.ID:
		return executor.FunctionSelectorBalancerV2.RawName
	case executor.FunctionSelectorDODO.ID:
		return executor.FunctionSelectorDODO.RawName
	case executor.FunctionSelectorGMX.ID:
		return executor.FunctionSelectorGMX.RawName
	case executor.FunctionSelectorSynthetix.ID:
		return executor.FunctionSelectorSynthetix.RawName
	case executor.FunctionSelectorPSM.ID:
		return executor.FunctionSelectorPSM.RawName
	case executor.FunctionSelectorWSTETH.ID:
		return executor.FunctionSelectorWSTETH.RawName
	case executor.FunctionSelectorSTETH.ID:
		return executor.FunctionSelectorSTETH.RawName
	case executor.FunctionSelectorKSClassic.ID:
		return executor.FunctionSelectorKSClassic.RawName
	case executor.FunctionSelectorVelodrome.ID:
		return executor.FunctionSelectorVelodrome.RawName
	case executor.FunctionSelectorPlatypus.ID:
		return executor.FunctionSelectorPlatypus.RawName
	case executor.FunctionSelectorFraxSwap.ID:
		return executor.FunctionSelectorFraxSwap.RawName
	case executor.FunctionSelectorCamelotSwap.ID:
		return executor.FunctionSelectorCamelotSwap.RawName
	case executor.FunctionSelectorLimitOrder.ID:
		return executor.FunctionSelectorLimitOrder.RawName
	case executor.FunctionSelectorLimitOrderDS.ID:
		return executor.FunctionSelectorLimitOrderDS.RawName
	case executor.FunctionSelectorTraderJoeV2.ID:
		return executor.FunctionSelectorTraderJoeV2.RawName
	case executor.FunctionSelectorKyberPMM.ID:
		return executor.FunctionSelectorKyberPMM.RawName
	default:
		return ""
	}
}

func (d *Decoder) decodeSwapData(sw executor.Swap) (interface{}, error) {
	functionSelector, _ := d.decodeSelectorAndFlags(sw.SelectorAndFlags)
	switch functionSelector {
	case executor.FunctionSelectorUniswap.ID:
		return swapdata.UnpackUniSwap(sw.Data)
	case executor.FunctionSelectorStableSwap.ID:
		return swapdata.UnpackStableSwap(sw.Data)
	case executor.FunctionSelectorCurveSwap.ID:
		return swapdata.UnpackCurveSwap(sw.Data)
	case executor.FunctionSelectorPancakeStableSwap.ID:
		return swapdata.UnpackPancakeStableSwap(sw.Data)
	case executor.FunctionSelectorUniV3KSElastic.ID:
		return swapdata.UnpackUniswapV3KSElastic(sw.Data)
	case executor.FunctionSelectorBalancerV2.ID:
		return swapdata.UnpackBalancerV2(sw.Data)
	case executor.FunctionSelectorDODO.ID:
		return swapdata.UnpackDODO(sw.Data)
	case executor.FunctionSelectorGMX.ID:
		return swapdata.UnpackGMX(sw.Data)
	case executor.FunctionSelectorSynthetix.ID:
		return swapdata.UnpackSynthetix(sw.Data)
	case executor.FunctionSelectorPSM.ID:
		return swapdata.UnpackPSM(sw.Data)
	case executor.FunctionSelectorWSTETH.ID:
		return swapdata.UnpackWSTETH(sw.Data)
	case executor.FunctionSelectorSTETH.ID:
		return swapdata.UnpackStETH(sw.Data)
	case executor.FunctionSelectorKSClassic.ID:
		return swapdata.UnpackUniSwap(sw.Data)
	case executor.FunctionSelectorVelodrome.ID:
		return swapdata.UnpackUniSwap(sw.Data)
	case executor.FunctionSelectorPlatypus.ID:
		return swapdata.UnpackPlatypus(sw.Data)
	case executor.FunctionSelectorFraxSwap.ID:
		return swapdata.UnpackUniSwap(sw.Data)
	case executor.FunctionSelectorCamelotSwap.ID:
		return swapdata.UnpackUniSwap(sw.Data)
	case executor.FunctionSelectorLimitOrder.ID:
		return swapdata.UnpackKyberLimitOrder(sw.Data)
	case executor.FunctionSelectorLimitOrderDS.ID:
		return swapdata.UnpackKyberLimitOrderDS(sw.Data)
	case executor.FunctionSelectorSyncSwap.ID:
		return swapdata.UnpackSyncSwap(sw.Data)
	case executor.FunctionSelectorMaverickV1.ID:
		return swapdata.UnpackMaverickV1(sw.Data)
	case executor.FunctionSelectorAlgebraV1.ID:
		return swapdata.UnpackAlgebraV1(sw.Data)
	case executor.FunctionSelectorTraderJoeV2.ID:
		return swapdata.UnpackTraderJoeV2(sw.Data)
	case executor.FunctionSelectorKyberPMM.ID:
		return swapdata.UnpackKyberRFQ(sw.Data)
	case executor.FunctionSelectorWombat.ID:
		return swapdata.UnpackWombat(sw.Data)
	case executor.FunctionSelectorVooi.ID:
		return swapdata.UnpackVooi(sw.Data)
	default:
		return nil, fmt.Errorf("unsupported function selector")
	}
}

func (d *Decoder) decodeSelectorAndFlags(sf executor.SwapSelectorAndFlags) (
	functionSelector executor.SwapSelector,
	flags executor.SwapFlags,
) {
	copy(functionSelector[:], sf[:len(functionSelector)])
	copy(flags[:], sf[len(sf)-len(flags):])
	// Reverse swap flags
	for i, j := 0, len(flags)-1; i < j; i, j = i+1, j-1 {
		flags[i], flags[j] = flags[j], flags[i]
	}
	return
}
