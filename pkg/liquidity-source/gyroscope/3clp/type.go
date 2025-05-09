package gyro3clp

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

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
	Vault           string `json:"vault"`
	PoolID          string `json:"poolId"`
	TokenOutIndex   int    `json:"tokenOutIndex"`
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress"`
}

type Extra struct {
	PoolTokenInfos    []PoolTokenInfo `json:"poolTokenInfos"`
	SwapFeePercentage *uint256.Int    `json:"swapFeePercentage"`
	Paused            bool            `json:"paused"`
}

type StaticExtra struct {
	PoolID         string         `json:"poolId"`
	PoolType       string         `json:"poolType"`
	PoolTypeVer    int            `json:"poolTypeVersion"`
	ScalingFactors []*uint256.Int `json:"scalingFactors"`
	Root3Alpha     *uint256.Int   `json:"root3Alpha"`
	Vault          string         `json:"vault"`
}

type PoolTokensResp struct {
	Tokens          []common.Address
	Balances        []*big.Int
	LastChangeBlock *big.Int
}

type PausedStateResp struct {
	Paused              bool
	PauseWindowEndTime  *big.Int
	BufferPeriodEndTime *big.Int
}

type PoolTokenInfoResp struct {
	Cash            *big.Int
	Managed         *big.Int
	LastChangeBlock *big.Int
	AssetManager    common.Address
}

type rpcRes struct {
	PoolTokens        PoolTokensResp
	PoolTokenInfos    []PoolTokenInfoResp
	SwapFeePercentage *big.Int
	PausedState       PausedStateResp
	BlockNumber       uint64
}
