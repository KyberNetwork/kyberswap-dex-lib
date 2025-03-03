package arbitrum

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/entity"
)

func EstimateL1Fees(params *entity.ArbitrumL1FeeParams) (*big.Int, *big.Int) {
	return calcL1Fee(params, l1GasOverhead), calcL1Fee(params, l1GasPerPool)
}

func calcL1Fee(params *entity.ArbitrumL1FeeParams, l1GasUsed *big.Int) (l1Fee *big.Int) {
	l1Fee = new(big.Int).Mul(params.L1BaseFee, l1GasUsed)
	return l1Fee
}
