package helper

import "github.com/KyberNetwork/router-service/internal/pkg/valueobject"

var l2EncoderSupportedChains = map[valueobject.ChainID]struct{}{
	valueobject.ChainIDArbitrumOne: {},
	valueobject.ChainIDOptimism:    {},
	valueobject.ChainIDBase:        {},
}

func IsL2EncoderSupportedChains(chainID valueobject.ChainID) bool {
	_, exist := l2EncoderSupportedChains[chainID]
	return exist
}
