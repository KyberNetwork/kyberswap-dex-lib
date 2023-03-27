package dto

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/valueobject"
)

type GetRoutesQuery struct {
	TokenIn  string
	TokenOut string
	AmountIn *big.Int

	IncludedSources []string
	ExcludedSources []string

	SaveGas    bool
	GasInclude bool
	GasPrice   *big.Float

	ExtraFee valueobject.ExtraFee
}
