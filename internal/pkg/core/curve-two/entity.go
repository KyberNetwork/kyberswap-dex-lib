package curveTwo

type Extra struct {
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
}

type Gas struct {
	Exchange int64
}
