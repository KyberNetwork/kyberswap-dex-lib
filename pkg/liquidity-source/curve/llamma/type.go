package llamma

import (
	"fmt"
	"math/big"

	"github.com/KyberNetwork/int256"
	"github.com/KyberNetwork/kutils"
	"github.com/holiman/uint256"
)

type BandResponse struct {
	Index      string `json:"index"`
	Collateral string `json:"collateral"`
	StableCoin string `json:"stableCoin"`
	ID         string `json:"id"`
}

type Band struct {
	Index      int64
	Collateral *uint256.Int
	StableCoin *uint256.Int
}

type FetchRPCResult struct {
	Fee                *big.Int
	AdminFee           *big.Int
	AdminFeesX         *big.Int
	AdminFeesY         *big.Int
	PriceOracle        *big.Int
	ActiveBand         *big.Int
	MinBand            *big.Int
	MaxBand            *big.Int
	BandsX             map[int64]*big.Int
	BandsY             map[int64]*big.Int
	CollateralReserves *big.Int
	StableCoinReserves *big.Int
	BlockNumber        *big.Int
}

type Extra struct {
	Fee         *uint256.Int           `json:"fee"`
	AdminFee    *uint256.Int           `json:"adminFee"`
	AdminFeesX  *uint256.Int           `json:"adminFeesX"`
	AdminFeesY  *uint256.Int           `json:"adminFeesY"`
	PriceOracle *uint256.Int           `json:"priceOracle"`
	ActiveBand  *int256.Int            `json:"activeBand"`
	MinBand     *int256.Int            `json:"minBand"`
	MaxBand     *int256.Int            `json:"maxBand"`
	BandsX      map[int64]*uint256.Int `json:"bandsX"`
	BandsY      map[int64]*uint256.Int `json:"bandsY"`
}

type DetailedTrade struct {
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

type StaticExtra struct {
	A         *uint256.Int
	BasePrice *uint256.Int
}

func (b *BandResponse) transformBandResponseToBand(collateralPrecision, stableCoinPrecision *big.Float) (Band, error) {
	collateralBF, ok := new(big.Float).SetString(b.Collateral)
	if !ok {
		return Band{}, fmt.Errorf("can not convert collateral string to float, tick: %v", b.Index)
	}
	stableCoinBF, ok := new(big.Float).SetString(b.StableCoin)
	if !ok {
		return Band{}, fmt.Errorf("can not convert stableCoin string to float, tick: %v", b.Index)
	}
	bandIndex, err := kutils.Atoi[int64](b.Index)
	if err != nil {
		return Band{}, fmt.Errorf("can not convert tickIdx string to int, tick: %v", b.Index)
	}

	collateralBI, _ := collateralBF.Mul(collateralBF, collateralPrecision).Int(nil)
	stableCoinBI, _ := stableCoinBF.Mul(stableCoinBF, stableCoinPrecision).Int(nil)

	return Band{
		Index:      bandIndex,
		Collateral: uint256.MustFromBig(collateralBI),
		StableCoin: uint256.MustFromBig(stableCoinBI),
	}, nil
}
