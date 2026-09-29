package velodromev1

type Config struct {
	DexID          string         `json:"dexID"`
	FactoryAddress string         `json:"factoryAddress"`
	NewPoolLimit   int            `json:"newPoolLimit"`
	AllPairs       uint32         `json:"allPairs"`
	AllPairsLength uint32         `json:"allPairsLength"`
	ManualMetadata bool           `json:"manualMetadata"`
	Stable         *bool          `json:"stable"` // with manualMetadata: fixed pool type for pools without stable()
	Fee            uint64         `json:"fee"`
	FeePrecision   uint64         `json:"feePrecision"`
	FeeTracker     *FeeTrackerCfg `json:"feeTracker"`
}

type FeeTrackerCfg struct {
	Target   string   `json:"target"`
	Selector uint32   `json:"selector"`
	Args     []string `json:"args"`
}
