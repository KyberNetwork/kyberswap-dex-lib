package limitorder

type ChainID uint

type tokenPair struct {
	Token0          string `json:"token0"`
	Token1          string `json:"token1"`
	ContractAddress string `json:"contractAddress"`
}

type Extra struct {
	SellOrders []*order
	BuyOrders  []*order
}

type StaticExtra struct {
	ContractAddress string
}

type SwapSide string

type SwapInfo struct {
	AmountIn     string             `json:"amountIn"`
	SwapSide     SwapSide           `json:"swapSide"`
	FilledOrders []*FilledOrderInfo `json:"filledOrders"`
}

type FilledOrderInfo struct {
	OrderID              int64  `json:"orderId"`
	OrderHash            string `json:"orderHash"`
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
	FeeConfig            string `json:"feeConfig"`
	FeeRecipient         string `json:"feeRecipient"`
	MakerTokenFeePercent uint32 `json:"makerTokenFeePercent"`
	MakerAssetData       string `json:"makerAssetData"`
	TakerAssetData       string `json:"takerAssetData"`
	Predicate            string `json:"predicate"`
	Permit               string `json:"permit"`
	Interaction          string `json:"interaction"`
	Signature            string `json:"signature"`
	IsFallBack           bool   `json:"isFallback"`

	// Quote-time making left in the order itself (before its maker's balance caps it) and its
	// fee side, so the RFQ handler can re-spread amountIn over the signed orders when a required
	// order loses its operator signature.
	RemainingMakingAmount string `json:"remainingMakingAmount,omitempty"`
	IsTakerAssetFee       bool   `json:"isTakerAssetFee,omitempty"`
	// MakerBalance is the maker's balance in the swap limit when the quote capped fills by it;
	// empty when the quote had no limit.
	MakerBalance string `json:"makerBalance,omitempty"`
}

type OpSignatureExtra struct {
	SwapInfo
	OperatorSignaturesById map[int64]*operatorSignatures `json:"operatorSignaturesById"`
}
