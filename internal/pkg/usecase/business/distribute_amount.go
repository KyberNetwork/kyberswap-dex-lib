package business

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

// DistributeAmount distributes amount following distributions
// the last chunk would be the rest of undistributed amount
func DistributeAmount(amount *big.Int, distributions []uint64) []*big.Int {
	undistributedAmount := new(big.Int).Set(amount)

	distributedAmounts := make([]*big.Int, 0, len(distributions))
	for i, distribution := range distributions {
		if i == len(distributions)-1 {
			distributedAmounts = append(distributedAmounts, undistributedAmount)
			continue
		}

		distributedAmount := new(big.Int).Div(
			new(big.Int).Mul(
				amount,
				new(big.Int).SetUint64(distribution),
			),
			valueobject.BasisPoint,
		)

		distributedAmounts = append(distributedAmounts, distributedAmount)
		undistributedAmount.Sub(undistributedAmount, distributedAmount)
	}

	return distributedAmounts
}

func CalcDistribution(amount *big.Int, pathAmount *big.Int) uint64 {
	distributionBigInt := new(big.Int).Mul(pathAmount, valueobject.BasisPoint)
	distributionBigInt.Div(distributionBigInt, amount)

	return distributionBigInt.Uint64()
}
