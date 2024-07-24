package dto

import (
	"math/big"

	mapset "github.com/deckarep/golang-set/v2"

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

	ExcludedPools mapset.Set[string]
	ClientId      string
}
