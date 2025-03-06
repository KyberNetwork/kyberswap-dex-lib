package llamma

import (
	"github.com/holiman/uint256"
)

type (
	StaticExtra struct {
		A             *uint256.Int `json:"A"`
		UseDynamicFee bool         `json:"useDynamicFee"`
	}

	Extra struct {
		BasePrice   *uint256.Int `json:"basePrice"`
		PriceOracle *uint256.Int `json:"priceOracle"`
		Fee         *uint256.Int `json:"fee"`
		AdminFee    *uint256.Int `json:"adminFee"`
		AdminFeesX  *uint256.Int `json:"adminFeesX"`
		AdminFeesY  *uint256.Int `json:"adminFeesY"`
		ActiveBand  int64        `json:"activeBand"`
		MinBand     int64        `json:"minBand"`
		MaxBand     int64        `json:"maxBand"`
		Bands       []Band       `json:"bands"`
	}

	Meta struct {
		TokenInIndex int    `json:"tokenInIndex"`
		BlockNumber  uint64 `json:"blockNumber"`
	}

	Band struct {
		Index int64        `json:"i"`
		BandX *uint256.Int `json:"x"`
		BandY *uint256.Int `json:"y"`
	}

	DetailedTrade struct {
		InAmount  uint256.Int   `json:"inAmount"`
		OutAmount uint256.Int   `json:"outAmount"`
		N1        int64         `json:"n1"`
		N2        int64         `json:"n2"`
		TicksIn   []uint256.Int `json:"ticksIn"`
		LastTickJ uint256.Int   `json:"lastTickJ"`
		AdminFee  uint256.Int   `json:"adminFee"`
	}

	Gas struct {
		Exchange int64
	}
)
