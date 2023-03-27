package uni

type Gas struct {
	SwapBase    int64
	SwapNonBase int64
}

type Meta struct {
	SwapFee string `json:"swapFee"`
}
