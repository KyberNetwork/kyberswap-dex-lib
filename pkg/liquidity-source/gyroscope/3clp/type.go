package gyro3clp

import "github.com/holiman/uint256"

type PoolTokenInfo struct {
	Cash            *uint256.Int `json:"cash"`
	Managed         *uint256.Int `json:"managed"`
	LastChangeBlock uint64       `json:"lastChangeBlock"`
	AssetManager    string       `json:"assetManager"`
}

type Gas struct {
	Swap int64
}

type PoolMetaInfo struct {
	Vault       string `json:"vault"`
	PoolID      string `json:"poolId"`
	T           string `json:"t"`
	V           int    `json:"v"`
	BlockNumber uint64 `json:"blockNumber"`
}
