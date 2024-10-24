package ringswap

type Extra struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`
}

type SwapInfo struct {
	IsToken0To1 bool `json:"isToken0To1"`
	IsWrapIn    bool `json:"isWrapIn"`
	IsUnwrapOut bool `json:"isUnwrapOut"`
}
