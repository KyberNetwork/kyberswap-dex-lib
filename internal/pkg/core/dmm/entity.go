package dmm

type PoolModelReserves []*string

type Extra struct {
	VReserves      PoolModelReserves `json:"vReserves"`
	FeeInPrecision string            `json:"feeInPrecision"`
}

type Gas struct {
	SwapBase    int64
	SwapNonBase int64
}
