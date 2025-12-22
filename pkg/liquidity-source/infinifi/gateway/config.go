package gateway

type Config struct {
	DexID   string `json:"dexId"`
	Gateway string `json:"gateway"`
	USDC    string `json:"usdc"`
	IUSD    string `json:"iusd"`
	SIUSD   string `json:"siusd"`
	// LockingController address to fetch liUSD token addresses
	LockingController string `json:"lockingController"`
}
