package client

import (
	"github.com/KyberNetwork/blockchain-toolkit/time/durationjson"
)

type HTTPConfig struct {
	BaseURL    string                `mapstructure:"base_url" json:"base_url,omitempty"`
	Timeout    durationjson.Duration `mapstructure:"timeout" json:"timeout,omitempty"`
	RetryCount int                   `mapstructure:"retry_count" json:"retry_count,omitempty"`
}

type PoolInfo struct {
	Fee            int    `json:"fee"`
	TokenX         string `json:"tokenX"`
	TokenY         string `json:"tokenY"`
	Address        string `json:"address"`
	Timestamp      int    `json:"timestamp"`
	TokenXAddress  string `json:"tokenX_address"`
	TokenYAddress  string `json:"tokenY_address"`
	TokenXDecimals int    `json:"tokenX_decimals"`
	TokenYDecimals int    `json:"tokenY_decimals"`
	Version        string `json:"version"`
}

type ListPoolsParams struct {
	ChainId int
	// v1 or v2
	Version string
	// timestamp in second
	TimeStart int
	// response size
	Limit int
}

type ListPoolsResponse struct {
	Data  []PoolInfo `json:"data,omitempty"`
	Total int        `json:"total"`
}
