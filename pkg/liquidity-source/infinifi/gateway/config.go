package gateway

type Config struct {
	DexID   string `json:"dexId"`
	Gateway string `json:"gateway"` // 0x3f04b65Ddbd87f9CE0A2e7Eb24d80e7fb87625b5
	USDC    string `json:"usdc"`
	IUSD    string `json:"iusd"`
	SIUSD   string `json:"siusd"`
	
	// Multiple liUSD tokens (one per unwinding epoch)
	// e.g. liUSD-3mo, liUSD-6mo, liUSD-12mo, etc.
	LIUSDTokens []LIUSDToken `json:"liusdTokens"`
	
	// LockingController address to fetch liUSD token addresses
	LockingController string `json:"lockingController"`
}

type LIUSDToken struct {
	Address         string `json:"address"`         // LockedPositionToken address
	UnwindingEpochs uint32 `json:"unwindingEpochs"` // 3, 6, 12, etc. (months)
	Name            string `json:"name"`            // e.g. "liUSD-12mo"
}

