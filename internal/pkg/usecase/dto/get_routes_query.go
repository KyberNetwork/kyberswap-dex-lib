package dto

import (
	"math/big"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
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

	IsPathGeneratorEnabled bool
}
