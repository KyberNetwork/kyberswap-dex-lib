package llamma

import (
	"github.com/KyberNetwork/int256"
	"github.com/holiman/uint256"
)

type (
	StaticExtra struct {
		A *uint256.Int `json:"A"`
	}

	Extra struct {
		BasePrice   *uint256.Int `json:"basePrice"`
		Fee         *uint256.Int `json:"fee"`
		AdminFeesX  *uint256.Int `json:"adminFeesX"`
		AdminFeesY  *uint256.Int `json:"adminFeesY"`
		AdminFee    *uint256.Int `json:"adminFee"`
		DynamicFee  *uint256.Int `json:"dynamicFee"`
		PriceOracle *uint256.Int `json:"priceOracle"`
		ActiveBand  *int256.Int  `json:"activeBand"`
		MinBand     *int256.Int  `json:"minBand"`
		MaxBand     *int256.Int  `json:"maxBand"`
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
		TokenInIdx   int
		InAmount     uint256.Int
		OutAmount    uint256.Int
		N1           int256.Int
		N2           int256.Int
		TicksIn      []uint256.Int
		LastTickJ    uint256.Int
		AdminFee     uint256.Int
		AdminFeeX    uint256.Int
		AdminFeeY    uint256.Int
		InPrecision  uint256.Int
		OutPrecision uint256.Int
	}
)
