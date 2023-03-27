package rest

var DefaultMaxPrice = "115792089237316195423570985008687907853269984665640564039457584007913129639935"

type Swap struct {
	Pool              string      `json:"pool"`
	TokenIn           string      `json:"tokenIn"`
	TokenOut          string      `json:"tokenOut"`
	SwapAmount        string      `json:"swapAmount"`
	AmountOut         string      `json:"amountOut"`
	LimitReturnAmount string      `json:"limitReturnAmount"`
	MaxPrice          string      `json:"maxPrice"`
	Exchange          string      `json:"exchange"`
	PoolLength        int         `json:"poolLength"`
	PoolType          string      `json:"poolType"`
	PoolExtra         interface{} `json:"poolExtra"`
	Extra             interface{} `json:"extra,omitempty"`
}
type TokenInfo struct {
	Address  string  `json:"address"`
	Symbol   string  `json:"symbol"`
	Name     string  `json:"name"`
	Price    float64 `json:"price"`
	Decimals uint8   `json:"decimals"`
}

type RouteResponse struct {
	InputAmount     string                 `json:"inputAmount"`
	OutputAmount    string                 `json:"outputAmount"`
	TotalGas        int64                  `json:"totalGas"`
	GasPriceGwei    string                 `json:"gasPriceGwei"`
	GasUsd          float64                `json:"gasUsd"`
	AmountInUsd     float64                `json:"amountInUsd"`
	AmountOutUsd    float64                `json:"amountOutUsd"`
	ReceivedUsd     float64                `json:"receivedUsd"`
	Swaps           [][]Swap               `json:"swaps"`
	Tokens          map[string]TokenInfo   `json:"tokens"`
	EncodedSwapData string                 `json:"encodedSwapData,omitempty"`
	RouterAddress   string                 `json:"routerAddress,omitempty"`
	Debug           map[string]interface{} `json:"debug,omitempty"`
}
