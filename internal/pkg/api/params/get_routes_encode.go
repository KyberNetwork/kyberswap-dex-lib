package params

type GetRouteEncodeParams struct {
	TokenIn              string     `form:"tokenIn" binding:"required"`
	TokenOut             string     `form:"tokenOut" binding:"required"`
	AmountIn             string     `form:"amountIn" binding:"required"`
	SaveGas              bool       `form:"saveGas"`
	Dexes                string     `form:"dexes"`
	OnlyScalableSources  bool       `form:"onlyScalableSources"`
	GasInclude           bool       `form:"gasInclude"`
	GasPrice             string     `form:"gasPrice"`
	SlippageTolerance    int64      `form:"slippageTolerance"`
	ChargeFeeBy          string     `form:"chargeFeeBy"`
	FeeReceiver          string     `form:"feeReceiver"`
	IsInBps              bool       `form:"isInBps"`
	FeeAmount            string     `form:"feeAmount"`
	Deadline             int64      `form:"deadline"`
	To                   string     `form:"to"`
	ClientData           ClientData `form:"clientData"`
	Referral             string     `form:"referral"`
	Permit               string     `form:"permit"`
	IgnoreCappedSlippage bool       `form:"ignoreCappedSlippage,default=false"`
}

type ClientData struct {
	Source string `json:"source"`
}

type (
	GetRouteEncodeResponse struct {
		InputAmount     string                                 `json:"inputAmount"`
		OutputAmount    string                                 `json:"outputAmount"`
		TotalGas        int64                                  `json:"totalGas"`
		GasPriceGwei    string                                 `json:"gasPriceGwei"`
		GasUsd          float64                                `json:"gasUsd"`
		AmountInUsd     float64                                `json:"amountInUsd"`
		AmountOutUsd    float64                                `json:"amountOutUsd"`
		ReceivedUsd     float64                                `json:"receivedUsd"`
		Swaps           [][]GetRouteEncodeResponseSwap         `json:"swaps"`
		Tokens          map[string]GetRouteEncodeResponseToken `json:"tokens"`
		EncodedSwapData string                                 `json:"encodedSwapData,omitempty"`
		RouterAddress   string                                 `json:"routerAddress,omitempty"`
	}

	GetRouteEncodeResponseSwap struct {
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
		MaxPrice          string      `json:"maxPrice"`
	}

	GetRouteEncodeResponseToken struct {
		Address  string  `json:"address"`
		Symbol   string  `json:"symbol"`
		Name     string  `json:"name"`
		Price    float64 `json:"price"`
		Decimals uint8   `json:"decimals"`
	}
)
