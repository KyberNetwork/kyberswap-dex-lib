package limitorder

type ChainID uint

type tokenPair struct {
	Token0 string `json:"token0"`
	Token1 string `json:"token1"`
}

type Extra struct {
	SellOrders []*order
	BuyOrders  []*order
}

type SwapSide string

type SwapInfo struct {
	AmountIn     string             `json:"amountIn"`
	SwapSide     SwapSide           `json:"swapSide"`
	FilledOrders []*FilledOrderInfo `json:"filledOrders"`
}

type FilledOrderInfo struct {
	OrderID              int64  `json:"orderId"`
	FilledTakingAmount   string `json:"filledTakingAmount"`
	FilledMakingAmount   string `json:"filledMakingAmount"`
	FeeAmount            string `json:"feeAmount"`
	TakingAmount         string `json:"takingAmount"`
	MakingAmount         string `json:"makingAmount"`
	Salt                 string `json:"salt"`
	MakerAsset           string `json:"makerAsset"`
	TakerAsset           string `json:"takerAsset"`
	Maker                string `json:"maker"`
	Receiver             string `json:"receiver"`
	AllowedSenders       string `json:"allowedSenders"`
	GetMakerAmount       string `json:"getMakerAmount"`
	GetTakerAmount       string `json:"getTakerAmount"`
	FeeRecipient         string `json:"feeRecipient"`
	MakerTokenFeePercent uint32 `json:"makerTokenFeePercent"`
	MakerAssetData       string `json:"makerAssetData"`
	TakerAssetData       string `json:"takerAssetData"`
	Predicate            string `json:"predicate"`
	Permit               string `json:"permit"`
	Interaction          string `json:"interaction"`
	Signature            string `json:"signature"`
	IsFallBack           bool   `json:"isFallback"`
}
