package encode

import (
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l1encode"
	"github.com/KyberNetwork/router-service/internal/pkg/usecase/encode/l2encode"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type EncoderFactory struct {
	config Config
}

func NewEncoderFactory(config Config) *EncoderFactory {
	return &EncoderFactory{config: config}
}

func (e *EncoderFactory) GetEncoder() IEncoder {
	if e.config.UseL2Optimize && IsL2EncoderSupportedChains(e.config.ChainID) {
		return l2encode.NewEncoder(l2encode.Config{
			RouterAddress:             e.config.RouterAddress,
			ExecutorAddress:           e.config.ExecutorAddress,
			ChainID:                   e.config.ChainID,
			IsPositiveSlippageEnabled: e.config.IsPositiveSlippageEnabled,
			MinimumPSThreshold:        e.config.MinimumPSThreshold,
			FunctionSelectorMappingID: e.config.FunctionSelectorMappingID,
		})
	}

	return l1encode.NewEncoder(l1encode.Config{
		RouterAddress:             e.config.RouterAddress,
		ExecutorAddress:           e.config.ExecutorAddress,
		ChainID:                   e.config.ChainID,
		IsPositiveSlippageEnabled: e.config.IsPositiveSlippageEnabled,
		MinimumPSThreshold:        e.config.MinimumPSThreshold,
	})
}

var l2EncoderSupportedChains = map[valueobject.ChainID]struct{}{
	valueobject.ChainIDArbitrumOne: {},
	valueobject.ChainIDOptimism:    {},
}

func IsL2EncoderSupportedChains(chainID valueobject.ChainID) bool {
	_, exist := l2EncoderSupportedChains[chainID]
	return exist
}
