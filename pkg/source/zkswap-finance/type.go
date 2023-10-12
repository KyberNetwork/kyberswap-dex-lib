package zkswapfinance

type Metadata struct {
	Offset int `json:"offset"`
}

type Gas struct {
	SwapBase    int64
	SwapNonBase int64
}

type Meta struct {
	SwapFee string `json:"swapFee"`
}
