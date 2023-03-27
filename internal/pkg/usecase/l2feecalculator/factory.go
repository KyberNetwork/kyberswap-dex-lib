package l2feecalculator

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/usecase"
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

func NewL2FeeCalculator(chainID valueobject.ChainID) usecase.IL2FeeCalculator {
	switch chainID {
	case valueobject.ChainIDOptimism:
		return NewOptimismFeeCalculator()

	case valueobject.ChainIDArbitrumOne:
		return NewArbitrumFeeCalculator()

	default:
		return nil
	}
}
