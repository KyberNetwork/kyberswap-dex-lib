package l1encode

import (
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/executor"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode/router"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/types"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type (
	encodeExecutorFunc = func(chainID valueobject.ChainID, routerAddress string, isPositiveSlippageEnabled bool, minimumPSThreshold int64, data types.EncodingData) ([]byte, error)
	encodeRouterFunc   = func(executorAddress string, executorData []byte, data types.EncodingData) ([]byte, error)
)

type Encoder struct {
	config                   Config
	encodeExecutorNormalMode encodeExecutorFunc
	encodeRouterNormalMode   encodeRouterFunc
	encodeExecutorSimpleMode encodeExecutorFunc
	encodeRouterSimpleMode   encodeRouterFunc
}

func NewEncoder(config Config) *Encoder {
	return &Encoder{
		config:                   config,
		encodeExecutorNormalMode: executor.BuildAndPackCallBytesInputs,
		encodeRouterNormalMode:   router.BuildAndPackSwapInputs,
		encodeExecutorSimpleMode: executor.BuildAndPackSimpleSwapData,
		encodeRouterSimpleMode:   router.BuildAndPackSwapSimpleModeInputs,
	}
}

func (e *Encoder) Encode(data types.EncodingData) (string, error) {
	encodeExecutor, encodeRouter := e.encodeExecutorNormalMode, e.encodeRouterNormalMode
	if data.EncodingMode.IsSimple() {
		encodeExecutor, encodeRouter = e.encodeExecutorSimpleMode, e.encodeRouterSimpleMode
	}

	executorData, err := encodeExecutor(e.config.ChainID, e.config.RouterAddress, e.config.IsPositiveSlippageEnabled, e.config.MinimumPSThreshold, data)
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
