package altfun

import (
	"time"

	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

const defaultAPIURL = "https://api.alt.fun/api/v1/tokens"

var defaultTimeout = durationjson.Duration{Duration: 15 * time.Second}

type HTTPConfig struct {
	Timeout    durationjson.Duration `json:"timeout"`
	RetryCount int                   `json:"retryCount,omitempty"`
}

type Config struct {
	DexID                string     `json:"dexID"`
	ZapAddress           string     `json:"zapAddress"`
	BondingAddress       string     `json:"bondingAddress"`
	FactoryAddress       string     `json:"factoryAddress"`
	GlobalStorageAddress string     `json:"globalStorageAddress"`
	APIURL               string     `json:"apiURL"`
	NewPoolLimit         int        `json:"newPoolLimit"`
	HTTPConfig           HTTPConfig `json:"httpConfig"`
}
