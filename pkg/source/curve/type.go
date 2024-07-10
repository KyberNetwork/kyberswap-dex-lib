package curve

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

type PoolAndRegistries struct {
	PoolAddress              common.Address
	RegistryOrFactoryABI     abi.ABI
	RegistryOrFactoryAddress string
}

type Metadata struct {
	MainRegistryOffset   int `json:"mainRegistryOffset"`
	MetaFactoryOffset    int `json:"metaFactoryOffset"`
	CryptoRegistryOffset int `json:"cryptoRegistryOffset"`
	CryptoFactoryOffset  int `json:"cryptoFactoryOffset"`
}

type PoolToken struct {
	Address   string `json:"address"`
	Precision string `json:"precision"`
	Rate      string `json:"rate"`
}

type PoolItem struct {
	ID               string      `json:"id"`
	Type             string      `json:"type"`
	Tokens           []PoolToken `json:"tokens"`
	LpToken          string      `json:"lpToken"`
	APrecision       string      `json:"aPrecision"`
	Version          int         `json:"version"`
	BasePool         string      `json:"basePool"`
	RateMultiplier   string      `json:"rateMultiplier"`
	UnderlyingTokens []string    `json:"underlyingTokens"`
}

type PoolMetaStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	BasePool             string   `json:"basePool"`
	RateMultiplier       string   `json:"rateMultiplier"`
	APrecision           string   `json:"aPrecision"`
	UnderlyingTokens     []string `json:"underlyingTokens"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
	Rates                []string `json:"rates"`
}

type PoolPlainOracleStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	APrecision           string   `json:"aPrecision"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
	Oracle               string   `json:"oracle"`
}

type PoolBaseStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	APrecision           string   `json:"aPrecision"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
	Rates                []string `json:"rates"`
}

type PoolAaveStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	UnderlyingTokens     []string `json:"underlyingTokens"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
}

type PoolCompoundStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	UnderlyingTokens     []string `json:"underlyingTokens"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
}

type PoolTwoStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
}

type PoolTricryptoStaticExtra struct {
	LpToken              string   `json:"lpToken"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
}

type PoolBaseExtra struct {
	InitialA     string `json:"initialA"`
	FutureA      string `json:"futureA"`
	InitialATime int64  `json:"initialATime"`
	FutureATime  int64  `json:"futureATime"`
	SwapFee      string `json:"swapFee"`
	AdminFee     string `json:"adminFee"`
}

type PoolPlainOracleExtra struct {
	Rates        []*big.Int `json:"rates"`
	InitialA     string     `json:"initialA"`
	FutureA      string     `json:"futureA"`
	InitialATime int64      `json:"initialATime"`
	FutureATime  int64      `json:"futureATime"`
	SwapFee      string     `json:"swapFee"`
	AdminFee     string     `json:"adminFee"`
}

type PoolMetaExtra struct {
	InitialA     string `json:"initialA"`
	FutureA      string `json:"futureA"`
	InitialATime int64  `json:"initialATime"`
	FutureATime  int64  `json:"futureATime"`
	SwapFee      string `json:"swapFee"`
	AdminFee     string `json:"adminFee"`
}

type PoolAaveExtra struct {
	InitialA            string `json:"initialA"`
	FutureA             string `json:"futureA"`
	InitialATime        int64  `json:"initialATime"`
	FutureATime         int64  `json:"futureATime"`
	SwapFee             string `json:"swapFee"`
	AdminFee            string `json:"adminFee"`
	OffpegFeeMultiplier string `json:"offpegFeeMultiplier"`
}

type PoolCompoundExtra struct {
	A        string   `json:"a"`
	SwapFee  string   `json:"swapFee"`
	AdminFee string   `json:"adminFee"`
	Rates    []string `json:"rates"`
}

type PoolTwoExtra struct {
	A                   string `json:"A"`
	D                   string `json:"D"`
	Gamma               string `json:"gamma"`
	PriceScale          string `json:"priceScale"`
	LastPrices          string `json:"lastPrices"`
	PriceOracle         string `json:"priceOracle"`
	FeeGamma            string `json:"feeGamma"`
	MidFee              string `json:"midFee"`
	OutFee              string `json:"outFee"`
	FutureAGammaTime    int64  `json:"futureAGammaTime"`
	FutureAGamma        string `json:"futureAGamma"`
	InitialAGammaTime   int64  `json:"initialAGammaTime"`
	InitialAGamma       string `json:"initialAGamma"`
	LastPricesTimestamp int64  `json:"lastPricesTimestamp"`
	LpSupply            string `json:"lpSupply"`
	XcpProfit           string `json:"xcpProfit"`
	VirtualPrice        string `json:"virtualPrice"`
	AllowedExtraProfit  string `json:"allowedExtraProfit"`
	AdjustmentStep      string `json:"adjustmentStep"`
	MaHalfTime          string `json:"maHalfTime"`
}

type PoolTricryptoExtra struct {
	A                   string   `json:"A"`
	D                   string   `json:"D"`
	Gamma               string   `json:"gamma"`
	PriceScale          []string `json:"priceScale"`
	LastPrices          []string `json:"lastPrices"`
	PriceOracle         []string `json:"priceOracle"`
	FeeGamma            string   `json:"feeGamma"`
	MidFee              string   `json:"midFee"`
	OutFee              string   `json:"outFee"`
	FutureAGammaTime    int64    `json:"futureAGammaTime"`
	FutureAGamma        string   `json:"futureAGamma"`
	InitialAGammaTime   int64    `json:"initialAGammaTime"`
	InitialAGamma       string   `json:"initialAGamma"`
	LastPricesTimestamp int64    `json:"lastPricesTimestamp"`
	LpSupply            string   `json:"lpSupply"`
	XcpProfit           string   `json:"xcpProfit"`
	VirtualPrice        string   `json:"virtualPrice"`
	AllowedExtraProfit  string   `json:"allowedExtraProfit"`
	AdjustmentStep      string   `json:"adjustmentStep"`
	MaHalfTime          string   `json:"maHalfTime"`
}

type Meta struct {
	TokenInIndex  int  `json:"tokenInIndex"`
	TokenOutIndex int  `json:"tokenOutIndex"`
	Underlying    bool `json:"underlying"`

	TokenInIsNative  *bool
	TokenOutIsNative *bool
}
