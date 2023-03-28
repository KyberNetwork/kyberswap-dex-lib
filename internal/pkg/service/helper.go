package service

import "github.com/KyberNetwork/router-service/internal/pkg/entity"

func ExtractPricesMapping(priceEntityByAddress map[string]entity.Price) map[string]float64 {
	prices := make(map[string]float64, len(priceEntityByAddress))
	for _, price := range priceEntityByAddress {
		prices[price.Address], _ = price.GetPreferredPrice()
	}

	return prices
}
