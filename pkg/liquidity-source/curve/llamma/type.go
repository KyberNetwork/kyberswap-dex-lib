package llamma

import (
	"github.com/holiman/uint256"
)

type (
	StaticExtra struct {
		A                  *uint256.Int `json:"A"`
		PriceOracleAddress string       `json:"priceOracleAddress"`
	}

	Extra struct {
		BasePrice   *uint256.Int `json:"basePrice"`
		Fee         *uint256.Int `json:"fee"`
		AdminFeesX  *uint256.Int `json:"adminFeesX"`
		AdminFeesY  *uint256.Int `json:"adminFeesY"`
		AdminFee    *uint256.Int `json:"adminFee"`
		DynamicFee  *uint256.Int `json:"dynamicFee"`
		PriceOracle *uint256.Int `json:"priceOracle"`
		ActiveBand  int64        `json:"activeBand"`
		MinBand     int64        `json:"minBand"`
		MaxBand     int64        `json:"maxBand"`
		Bands       []Band       `json:"bands"`
	}

	Band struct {
		Index int64        `json:"i"`
		BandX *uint256.Int `json:"x"`
		BandY *uint256.Int `json:"y"`
	}
)

type (
	DetailedTrade struct {
		TokenInIndex int           `json:"tokenInIndex"`
		InAmount     uint256.Int   `json:"-"`
		OutAmount    uint256.Int   `json:"-"`
		N1           int64         `json:"n1"`
		N2           int64         `json:"n2"`
		TicksIn      []uint256.Int `json:"ticksIn"`
		LastTickJ    uint256.Int   `json:"lastTickJ"`
		AdminFee     uint256.Int   `json:"adminFee"`
		AdminFeeX    uint256.Int   `json:"adminFeeX"`
		AdminFeeY    uint256.Int   `json:"adminFeeY"`
		InPrecision  *uint256.Int  `json:"-"`
		OutPrecision *uint256.Int  `json:"-"`
	}
)
