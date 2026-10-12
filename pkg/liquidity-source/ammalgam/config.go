package ammalgam

import "time"

type Config struct {
	DexID              string        `json:"dexID"`
	FactoryAddress     string        `json:"factoryAddress"`
	PairBeacon         string        `json:"pairBeacon,omitempty"`
	PairImplementation string        `json:"pairImplementation,omitempty"`
	NewPoolLimit       int           `json:"newPoolLimit"`
	Gas                int64         `json:"gas,omitempty"`
	RefreshInterval    time.Duration `json:"refreshInterval,omitempty"`
}
