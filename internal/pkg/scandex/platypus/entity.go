package platypus

import (
	"encoding/json"
	"math/big"
	"strings"

	"github.com/KyberNetwork/router-service/internal/pkg/constant"
	"github.com/KyberNetwork/router-service/internal/pkg/utils/duration"
)

type Properties struct {
	SubgraphAPI           string
	GetPoolAddressesBulk  int
	NewPoolJobIntervalSec int
	ReserveJobInterval    duration.Duration `json:"reserveJobInterval"`
}

func NewProperties(data map[string]interface{}) (properties Properties, err error) {
	bodyBytes, _ := json.Marshal(data)
	err = json.Unmarshal(bodyBytes, &properties)

	return
}

type StaticExtra struct {
	PriceOracle string     `json:"priceOracle"`
	OracleType  OracleType `json:"oracleType"`
}

func NewStaticExtra(poolState PoolState) StaticExtra {
	priceOracle := strings.ToLower(poolState.PriceOracle.Hex())

	var oracleType OracleType
	switch priceOracle {
	case constant.AddressZero:
		oracleType = OracleTypeNone
	case AddressStakedAvax:
		oracleType = OracleTypeStakedAvax
	default:
		oracleType = OracleTypeChainlink
	}

	return StaticExtra{
		PriceOracle: priceOracle,
		OracleType:  oracleType,
	}
}

type Extra struct {
	C1             *big.Int         `json:"c1"`
	HaircutRate    *big.Int         `json:"haircutRate"`
	RetentionRatio *big.Int         `json:"retentionRatio"`
	SlippageParamK *big.Int         `json:"slippageParamK"`
	SlippageParamN *big.Int         `json:"slippageParamN"`
	XThreshold     *big.Int         `json:"xThreshold"`
	Paused         bool             `json:"paused"`
	SAvaxRate      *big.Int         `json:"sAvaxRate"`
	AssetByToken   map[string]Asset `json:"assetByToken"`
}

type Asset struct {
	Address          string   `json:"address"`
	Decimals         uint8    `json:"decimals"`
	Cash             *big.Int `json:"cash"`
	Liability        *big.Int `json:"liability"`
	UnderlyingToken  string   `json:"underlyingToken"`
	AggregateAccount string   `json:"aggregateAccount"`
}

func NewExtra(poolState PoolState, assetStates []AssetState, sAvaxRate *big.Int) Extra {
	assetByToken := make(map[string]Asset, len(assetStates))
	for _, assetState := range assetStates {
		assetByToken[strings.ToLower(assetState.UnderlyingToken.Hex())] = Asset{
			Address:          assetState.Address,
			Decimals:         assetState.Decimals,
			Cash:             assetState.Cash,
			Liability:        assetState.Liability,
			UnderlyingToken:  strings.ToLower(assetState.UnderlyingToken.Hex()),
			AggregateAccount: strings.ToLower(assetState.AggregateAccount.Hex()),
		}
	}

	return Extra{
		C1:             poolState.C1,
		HaircutRate:    poolState.HaircutRate,
		RetentionRatio: poolState.RetentionRatio,
		SlippageParamK: poolState.SlippageParamK,
		SlippageParamN: poolState.SlippageParamN,
		XThreshold:     poolState.XThreshold,
		Paused:         poolState.Paused,
		AssetByToken:   assetByToken,
		SAvaxRate:      sAvaxRate,
	}
}

type OracleType string

const (
	OracleTypeNone       = "None"
	OracleTypeChainlink  = "Chainlink"
	OracleTypeStakedAvax = "StakedAvax"
)
