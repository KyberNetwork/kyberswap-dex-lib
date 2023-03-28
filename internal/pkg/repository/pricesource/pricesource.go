package pricesource

import (
	"github.com/KyberNetwork/router-service/internal/pkg/repository"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/pricesource/coingecko"
	"github.com/KyberNetwork/router-service/internal/pkg/repository/pricesource/krystal"
)

type TypePriceSource string

const (
	TypeCoingecko TypePriceSource = "Coingecko"
	TypeKrystal   TypePriceSource = "Krystal"
)

func NewPriceSource(sourceType TypePriceSource) repository.IPriceSourceRepository {
	switch sourceType {
	case TypeCoingecko:
		return coingecko.NewCoingeckoPriceSource(coingecko.APIEndpoint, coingecko.TimeoutLong)

	case TypeKrystal:
		return krystal.NewKrystalPriceSource(krystal.APIEndpoint, krystal.TimeoutLong)

	default:
		return coingecko.NewCoingeckoPriceSource(coingecko.APIEndpoint, coingecko.TimeoutLong)
	}
}
