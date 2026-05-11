package stable

type HookExtra struct {
	Balances            []string `json:"balances"`
	Rates               []string `json:"rates"`
	LpSupply            string   `json:"lpSupply"`
	InitialA            string   `json:"initialA"`
	FutureA             string   `json:"futureA"`
	InitialATime        int64    `json:"initialATime"`
	FutureATime         int64    `json:"futureATime"`
	SwapFee             string   `json:"swapFee"`
	AdminFee            string   `json:"adminFee"`
	OffpegFeeMultiplier string   `json:"offpegFeeMultiplier"`
}
