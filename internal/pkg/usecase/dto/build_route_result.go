package dto

type BuildRouteResult struct {
	AmountIn    string `json:"amountIn"`
	AmountInUSD string `json:"amountInUsd"`

	AmountOut    string `json:"amountOut"`
	AmountOutUSD string `json:"amountOutUsd"`

	Gas    string `json:"gas"`
	GasUSD string `json:"gasUsd"`

	OutputChange OutputChange `json:"outputChange"`

	Data          string `json:"data"`
	RouterAddress string `json:"routerAddress"`
}

type OutputChange struct {
	Amount  string            `json:"amount"`
	Percent float64           `json:"percent"`
	Level   OutputChangeLevel `json:"level"`
}

type OutputChangeLevel int

const (
	OutputChangeLevelNormal  OutputChangeLevel = 0
	OutputChangeLevelWarning OutputChangeLevel = 1
	OutputChangeLevelFatal   OutputChangeLevel = 2
)
