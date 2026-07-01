package machima

import "net/http"

type Config struct {
	DexID           string      `json:"dexID"`
	SubgraphAPI     string      `json:"subgraphAPI,omitempty"`
	SubgraphHeaders http.Header `json:"subgraphHeaders,omitempty"`
	FactoryAddress  string      `json:"factoryAddress"`
	ClankNow        string      `json:"clankNow"`
	SwapAdapter     string      `json:"swapAdapter"`
	TickLensAddress string      `json:"tickLensAddress"`
	RouterAddress   string      `json:"routerAddress"`
	ChainID         uint64      `json:"chainID"`
	// Counter assets
	WETH string `json:"weth"`
	USDC string `json:"usdc"`
	XMA  string `json:"xma"`
}
