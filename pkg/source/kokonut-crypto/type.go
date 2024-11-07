package kokonutcrypto

type Metadata struct {
	Offset int `json:"offset"`
}

type StaticExtra struct {
	LpToken              string   `json:"lpToken"`
	PrecisionMultipliers []string `json:"precisionMultipliers"`
}

type Extra struct {
	A                              string `json:"A"`
	D                              string `json:"D"`
	Gamma                          string `json:"gamma"`
	PriceScale                     string `json:"priceScale"`
	LastPrices                     string `json:"lastPrices"`
	PriceOracle                    string `json:"priceOracle"`
	FeeGamma                       string `json:"feeGamma"`
	MidFee                         string `json:"midFee"`
	OutFee                         string `json:"outFee"`
	FutureAGammaTime               int64  `json:"futureAGammaTime"`
	FutureA                        string `json:"futureA"`
	FutureGamma                    string `json:"futureGamma"`
	InitialAGammaTime              int64  `json:"initialAGammaTime"`
	InitialA                       string `json:"initialA"`
	InitialGamma                   string `json:"initialGamma"`
	LastPricesTimestamp            int64  `json:"lastPricesTimestamp"`
	LpSupply                       string `json:"lpSupply"`
	XcpProfit                      string `json:"xcpProfit"`
	VirtualPrice                   string `json:"virtualPrice"`
	AllowedExtraProfit             string `json:"allowedExtraProfit"`
	AdjustmentStep                 string `json:"adjustmentStep"`
	MaHalfTime                     string `json:"maHalfTime"`
	MinRemainingPostRebalanceRatio string `json:"minRemainingPostRebalanceRatio"`
}
