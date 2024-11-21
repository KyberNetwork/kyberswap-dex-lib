package mxtrading

type (
	OrderParams struct {
		BaseToken  string `json:"baseToken"`
		QuoteToken string `json:"quoteToken"`
		Amount     string `json:"amount"`
		Taker      string `json:"taker"`
		FeeBps     uint   `json:"feeBps"`
	}

	Order struct {
		MakerAsset   string `json:"makerAsset"`
		TakerAsset   string `json:"takerAsset"`
		MakingAmount string `json:"makingAmount"`
		TakingAmount string `json:"takingAmount"`
		Maker        string `json:"maker"`
		Salt         string `json:"salt"`
		Receiver     string `json:"receiver"`
		MakerTraits  string `json:"makerTraits"`
	}

	SignedOrderResult struct {
		Order     *Order `json:"order"`
		Signature string `json:"signature"`
	}
)

type (
	PriceLevel struct {
		Size  float64 `json:"s"`
		Price float64 `json:"p"`
	}

	PoolExtra struct {
		ZeroToOnePriceLevels []PriceLevel `json:"0to1"`
		OneToZeroPriceLevels []PriceLevel `json:"1to0"`
	}

	SwapInfo struct {
		BaseToken       string `json:"b"`
		BaseTokenAmount string `json:"bAmt"`
		QuoteToken      string `json:"q"`
	}

	Gas struct {
		FillOrderArgs int64
	}

	MetaInfo struct {
		Timestamp int64 `json:"timestamp"`
	}
)

type RFQExtra struct {
	Router    string `json:"router"`
	Recipient string `json:"recipient"`
	SignedOrderResult
}
