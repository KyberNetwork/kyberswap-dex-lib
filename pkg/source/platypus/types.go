package platypus

import (
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	DexID       string `json:"dexID"`
	SubgraphAPI string `json:"subgraphAPI"`
}

type SubgraphPool struct {
	ID         string `json:"id"`
	LastUpdate string `json:"lastUpdate"`
}

type Metadata struct {
	LastUpdate string `json:"lastUpdate"`
}

type Asset struct {
	Address          string   `json:"address"`
	Decimals         uint8    `json:"decimals"`
	Cash             *big.Int `json:"cash"`
	Liability        *big.Int `json:"liability"`
	UnderlyingToken  string   `json:"underlyingToken"`
	AggregateAccount string   `json:"aggregateAccount"`
}

type Extra struct {
	PriceOracle    string           `json:"priceOracle"`
	OracleType     string           `json:"oracleType"`
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

func newExtra(poolState PoolState, assetStates []AssetState, sAvaxRate *big.Int) Extra {
	assetByToken := make(map[string]Asset, len(assetStates))
	for _, assetState := range assetStates {
		token := strings.ToLower(assetState.UnderlyingToken.Hex())
		assetByToken[token] = Asset{
			Address:          assetState.Address,
			Decimals:         assetState.Decimals,
			Cash:             assetState.Cash,
			Liability:        assetState.Liability,
			UnderlyingToken:  token,
			AggregateAccount: strings.ToLower(assetState.AggregateAccount.Hex()),
		}
	}

	priceOracle := strings.ToLower(poolState.PriceOracle.Hex())
	extra := Extra{
		PriceOracle:    priceOracle,
		OracleType:     getOracleType(priceOracle),
		C1:             poolState.C1,
		HaircutRate:    poolState.HaircutRate,
		RetentionRatio: poolState.RetentionRatio,
		SlippageParamK: poolState.SlippageParamK,
		SlippageParamN: poolState.SlippageParamN,
		XThreshold:     poolState.XThreshold,
		Paused:         poolState.Paused,
		AssetByToken:   assetByToken,
	}

	poolType := getPoolTypeByPriceOracle(priceOracle)
	if poolType == poolTypePlatypusAvax {
		extra.SAvaxRate = sAvaxRate
	}

	return extra
}

// PoolState represents data of pool smart contract
type PoolState struct {
	Address        string
	C1             *big.Int
	HaircutRate    *big.Int
	PriceOracle    common.Address
	RetentionRatio *big.Int
	SlippageParamK *big.Int
	SlippageParamN *big.Int
	TokenAddresses []common.Address
	XThreshold     *big.Int
	Paused         bool

	Type string
}

// AssetState represents data of asset smart contract
type AssetState struct {
	Address          string
	Decimals         uint8
	Cash             *big.Int
	Liability        *big.Int
	UnderlyingToken  common.Address
	AggregateAccount common.Address
}

type Gas struct {
	Swap int64
}
