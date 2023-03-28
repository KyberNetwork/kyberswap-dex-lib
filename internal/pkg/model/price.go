package model

import (
	"strconv"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/utils"
)

const PriceKey = "prices"

type Price struct {
	Address     string
	Price       float64
	Liquidity   float64
	LpAddress   string
	MarketPrice float64
}

func (p Price) EncodePrice() string {
	return utils.Join(p.Price, p.Liquidity, p.LpAddress, p.MarketPrice)
}

func DecodePrice(key, member string) Price {
	var p Price
	split := strings.Split(member, ":")
	p.Address = key
	p.Price, _ = strconv.ParseFloat(split[0], 64)
	p.Liquidity, _ = strconv.ParseFloat(split[1], 64)
	p.LpAddress = split[2]
	if len(split) == 4 {
		p.MarketPrice, _ = strconv.ParseFloat(split[3], 64)
	} else {
		p.MarketPrice = 0
	}
	return p
}
