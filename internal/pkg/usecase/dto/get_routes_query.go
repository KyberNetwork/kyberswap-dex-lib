package dto

import (
	"math/big"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type GetRoutesQuery struct {
	TokenIn  string
	TokenOut string
	AmountIn *big.Int

	IncludedSources     []string
	ExcludedSources     []string
	OnlyScalableSources bool

	SaveGas        bool
	OnlySinglePath bool
	GasInclude     bool
	GasPrice       *big.Float

	ExtraFee valueobject.ExtraFee

	ExcludedPools mapset.Set[string]
	ClientId      string

	Index string
}

type GetBundledRoutesQueryPair struct {
	TokenIn  string
	TokenOut string
	AmountIn *big.Int
}

type GetBundledRoutesQuery struct {
	Pairs []*GetBundledRoutesQueryPair

	IncludedSources     []string
	ExcludedSources     []string
	OnlyScalableSources bool

	SaveGas    bool
	GasInclude bool
	GasPrice   *big.Float

	ExcludedPools mapset.Set[string]
	ClientId      string

	OverridePools json.RawMessage

	Index string
}
