package weighted

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Extra struct {
	SwapFeePercentage         *uint256.Int `json:"swapFeePercentage"`
	ProtocolSwapFeePercentage *uint256.Int `json:"protocolSwapFeePercentage"`
	LastInvariant             *uint256.Int `json:"lastInvariant"`
	TotalSupply               *uint256.Int `json:"totalSupply"`
	Paused                    bool         `json:"paused"`
}

type StaticExtra struct {
	PoolID                string              `json:"poolId"`
	PoolType              string              `json:"poolType"`
	PoolTypeVer           int                 `json:"poolTypeVer"`
	ScalingFactors        []*uint256.Int      `json:"scalingFactors"`
	NormalizedWeights     []*uint256.Int      `json:"normalizedWeights"`
	Vault                 string              `json:"vault"`
	BasePoolScanned       bool                `json:"basePoolScanned"`
	BatchSwapEnabled      bool                `json:"batchSwapEnabled"`
	ProtocolFeesCollector string              `json:"protocolFeesCollector"`
	BasePools             map[string][]string `json:"basePools,omitempty"`
}

type PoolTokens struct {
	Tokens          []common.Address
	Balances        []*big.Int
	LastChangeBlock *big.Int
}

type PausedState struct {
	Paused              bool
	PauseWindowEndTime  *big.Int
	BufferPeriodEndTime *big.Int
}

type PoolMetaInfo struct {
	Vault           string `json:"vault"`
	PoolID          string `json:"poolId"`
	TokenOutIndex   int    `json:"tokenOutIndex"`
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress"`
}

type rpcRes struct {
	PoolTokens                PoolTokens
	SwapFeePercentage         *uint256.Int
	ProtocolSwapFeePercentage *uint256.Int
	PausedState               PausedState
	LastInvariant             *uint256.Int
	TotalSupply               *uint256.Int
	BlockNumber               uint64
}
