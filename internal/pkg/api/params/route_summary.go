package params

type (
	RouteSummary struct {
		TokenIn     string `json:"tokenIn"`
		AmountIn    string `json:"amountIn"`
		AmountInUSD string `json:"amountInUsd"`

		TokenOut     string `json:"tokenOut"`
		AmountOut    string `json:"amountOut"`
		AmountOutUSD string `json:"amountOutUsd"`

		Gas      string `json:"gas"`
		GasPrice string `json:"gasPrice"`
		GasUSD   string `json:"gasUsd"`
		L1FeeUSD string `json:"l1FeeUsd"`

		ExtraFee ExtraFee `json:"extraFee"`

		Route [][]Swap `json:"route"`

		// Alpha fee
		AlphaFee *AlphaFeeV2 `json:"alphaFee,omitempty"`

		RouteSummaryValidation
	}

	AlphaFeeV2 struct {
		AMMAmount      string                    `json:"ammAmount"`
		SwapReductions []AlphaFeeV2SwapReduction `json:"swapReductions"`
	}
	AlphaFeeV2SwapReduction struct {
		ExecutedId      int
		Token           string
		ReduceAmount    string
		ReduceAmountUsd float64
	}

	RouteSummaryValidation struct {
		RouteID   string `json:"routeID"`
		Checksum  string `json:"checksum"`
		Timestamp int64  `json:"timestamp"`
	}

	ExtraFee struct {
		FeeAmount   string `json:"feeAmount"`
		ChargeFeeBy string `json:"chargeFeeBy"`
		IsInBps     bool   `json:"isInBps"`
		FeeReceiver string `json:"feeReceiver"`
	}

	Swap struct {
		Pool       string      `json:"pool"`
		TokenIn    string      `json:"tokenIn"`
		TokenOut   string      `json:"tokenOut"`
		SwapAmount string      `json:"swapAmount"`
		AmountOut  string      `json:"amountOut"`
		Exchange   string      `json:"exchange"`
		PoolType   string      `json:"poolType"`
		PoolExtra  interface{} `json:"poolExtra"`
		Extra      interface{} `json:"extra"`
	}

	ChunkInfo struct {
		AmountIn     string `json:"amountIn"`
		AmountOut    string `json:"amountOut"`
		AmountInUSD  string `json:"amountInUsd"`
		AmountOutUSD string `json:"amountOutUsd"`
	}

	RouteExtraData struct {
		ChunksInfo []ChunkInfo `json:"chunksInfo"`
	}
)
