package ringswap

type StaticExtra struct {
	Fee          uint64 `json:"fee"`
	FeePrecision uint64 `json:"feePrecision"`
}

type SwapInfo struct {
	WTokenIn    string `json:"wTokenIn"`
	WTokenOut   string `json:"wTokenOut"`
	IsToken0To1 bool   `json:"isToken0To1"`
	IsWrapIn    bool   `json:"isWrapIn"`
	IsUnwrapOut bool   `json:"isUnwrapOut"`
}
