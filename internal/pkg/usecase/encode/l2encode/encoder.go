package l2encode

import (
	l1router "github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/router"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode/executor"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type (
	encodeExecutorFunc = func(
		chainID valueobject.ChainID,
		routerAddress, executorAddress string,
		functionSelectorMappingID map[string]byte,
		isPositiveSlippageEnabled bool,
		minimumPSAmount int64,
		data types.EncodingData,
	) ([]byte, error)

	encodeRouterFunc = func(executorAddress string, executorData []byte, data types.EncodingData) ([]byte, error)

	Encoder struct {
		config                   Config
		encodeExecutorNormalMode encodeExecutorFunc
		encodeRouterNormalMode   encodeRouterFunc
		encodeExecutorSimpleMode encodeExecutorFunc
		encodeRouterSimpleMode   encodeRouterFunc
	}
)

func NewEncoder(config Config) *Encoder {
	return &Encoder{
		config:                   config,
		encodeExecutorNormalMode: executor.PackCallBytesInputs,
		// encodeRouterNormalMode:   router.PackSwapInputs,
		encodeRouterNormalMode: l1router.BuildAndPackSwapInputs,

		encodeExecutorSimpleMode: executor.PackSimpleSwapData,
		// encodeRouterSimpleMode:   router.PackSwapSimpleModeInputs,
		encodeRouterSimpleMode: l1router.BuildAndPackSwapSimpleModeInputs,
	}
}

func (e *Encoder) Encode(data types.EncodingData) (string, error) {
	encodeExecutor, encodeRouter := e.encodeExecutorNormalMode, e.encodeRouterNormalMode
	if data.EncodingMode.IsSimple() {
		encodeExecutor, encodeRouter = e.encodeExecutorSimpleMode, e.encodeRouterSimpleMode
	}

	executorData, err := encodeExecutor(
		e.config.ChainID,
		e.config.RouterAddress,
		e.config.ExecutorAddress,
		e.config.FunctionSelectorMappingID,
		e.config.IsPositiveSlippageEnabled,
		e.config.MinimumPSThreshold,
		data,
	)
	if err != nil {
		return "", err
	}
	routerData, err := encodeRouter(e.config.ExecutorAddress, executorData, data)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(routerData), nil
}

func (e *Encoder) GetExecutorAddress() string {
	return e.config.ExecutorAddress
}

func (e *Encoder) GetRouterAddress() string {
	return e.config.RouterAddress
}
