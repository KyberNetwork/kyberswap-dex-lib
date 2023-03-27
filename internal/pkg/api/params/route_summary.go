package params

type (
	RouteSummary struct {
		TokenIn                     string `json:"tokenIn"`
		AmountIn                    string `json:"amountIn"`
		AmountInUSD                 string `json:"amountInUsd"`
		TokenInMarketPriceAvailable bool   `json:"tokenInMarketPriceAvailable"`

		TokenOut                     string `json:"tokenOut"`
		AmountOut                    string `json:"amountOut"`
		AmountOutUSD                 string `json:"amountOutUsd"`
		TokenOutMarketPriceAvailable bool   `json:"tokenOutMarketPriceAvailable"`

		Gas      string `json:"gas"`
		GasPrice string `json:"gasPrice"`
		GasUSD   string `json:"gasUsd"`

		ExtraFee ExtraFee `json:"extraFee"`

		Route [][]Swap `json:"route"`
	}

	ExtraFee struct {
		FeeAmount   string `json:"feeAmount"`
		ChargeFeeBy string `json:"chargeFeeBy"`
		IsInBps     bool   `json:"isInBps"`
		FeeReceiver string `json:"feeReceiver"`
	}

	Swap struct {
		Pool              string      `json:"pool"`
		TokenIn           string      `json:"tokenIn"`
		TokenOut          string      `json:"tokenOut"`
		LimitReturnAmount string      `json:"limitReturnAmount"`
		SwapAmount        string      `json:"swapAmount"`
		AmountOut         string      `json:"amountOut"`
		Exchange          string      `json:"exchange"`
		PoolLength        int         `json:"poolLength"`
		PoolType          string      `json:"poolType"`
		PoolExtra         interface{} `json:"poolExtra"`
		Extra             interface{} `json:"extra"`
	}
)
