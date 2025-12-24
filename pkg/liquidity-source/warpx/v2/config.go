package warpx

type Config struct {
	DexID          string         `json:"dexID"`
	FactoryAddress string         `json:"factoryAddress"`
	RouterAddress  string         `json:"routerAddress"`
	Fee            uint64         `json:"fee"`
	FeePrecision   uint64         `json:"feePrecision"`
	FeeTracker     *FeeTrackerCfg `json:"feeTracker"`
	NewPoolLimit   int            `json:"newPoolLimit"`
}

type FeeTrackerCfg struct {
	Target   string   `json:"target"`
	Selector uint32   `json:"selector"`
	Args     []string `json:"args"`
}
