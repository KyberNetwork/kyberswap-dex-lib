package gateway

type Config struct {
	DexID   string `json:"dexId"`
	Gateway string `json:"gateway"`
	USDC    string `json:"usdc"`
	IUSD    string `json:"iusd"`
	SIUSD   string `json:"siusd"`
	
	// Multiple liUSD tokens (one per unwinding epoch)
	LIUSDTokens []LIUSDToken `json:"liusdTokens"`
	
	// LockingController address to fetch liUSD token addresses
	LockingController string `json:"lockingController"`
}

type LIUSDToken struct {
	Address         string `json:"address"`         // LockedPositionToken address
	UnwindingEpochs uint32 `json:"unwindingEpochs"`
	Name            string `json:"name"`
}

