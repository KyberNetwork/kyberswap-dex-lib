package composablestable

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type PoolMetaInfo struct {
	Vault       string `json:"vault"`
	PoolID      string `json:"poolId"`
	T           string `json:"t"`
	V           int    `json:"v"`
	BlockNumber uint64 `json:"blockNumber"`
}

type SwapInfo struct {
	LastJoinExitData LastJoinExitData `json:"-"`
}

type LastJoinExitData struct {
	LastJoinExitAmplification *uint256.Int `json:"lastJoinExitAmplification"`
	LastPostJoinExitInvariant *uint256.Int `json:"lastPostJoinExitInvariant"`
}

type TokenRateCache struct {
	Rate     *uint256.Int `json:"rate"`
	OldRate  *uint256.Int `json:"oldRate"`
	Duration *uint256.Int `json:"duration"`
	Expires  *uint256.Int `json:"expires"`
}

type Gas struct {
	Swap int64
}

type Extra struct {
	CanNotUpdateTokenRates            bool                 `json:"canNotUpdateTokenRates"`
	ScalingFactors                    []*uint256.Int       `json:"scalingFactors"`
	BptTotalSupply                    *uint256.Int         `json:"bptTotalSupply"`
	Amp                               *uint256.Int         `json:"amp"`
	LastJoinExit                      LastJoinExitData     `json:"lastJoinExit"`
	RateProviders                     []string             `json:"rateProviders"`
	TokenRateCaches                   []TokenRateCache     `json:"tokenRateCaches"`
	SwapFeePercentage                 *uint256.Int         `json:"swapFeePercentage"`
	ProtocolFeePercentageCache        map[int]*uint256.Int `json:"protocolFeePercentageCache"`
	IsTokenExemptFromYieldProtocolFee []bool               `json:"isTokenExemptFromYieldProtocolFee"`
	IsExemptFromYieldProtocolFee      bool                 `json:"isExemptFromYieldProtocolFee"`
	InRecoveryMode                    bool                 `json:"inRecoveryMode"`
	Paused                            bool                 `json:"paused"`
}

type StaticExtra struct {
	PoolID         string         `json:"poolId"`
	PoolType       string         `json:"poolType"`
	PoolTypeVer    int            `json:"poolTypeVer"`
	BptIndex       int            `json:"bptIndex"`
	ScalingFactors []*uint256.Int `json:"scalingFactors"`
	Vault          string         `json:"vault"`
}

type AmplificationParameterResp struct {
	Value      *big.Int
	IsUpdating bool
	Precision  *big.Int
}

type LastJoinExitResp struct {
	LastJoinExitAmplification *big.Int
	LastPostJoinExitInvariant *big.Int
}

type TokenRateCacheResp struct {
	Rate     *big.Int
	OldRate  *big.Int
	Duration *big.Int
	Expires  *big.Int
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

type rpcRes struct {
	CanNotUpdateTokenRates            bool
	PoolTokens                        PoolTokensResp
	BptTotalSupply                    *big.Int
	Amp                               *big.Int
	LastJoinExit                      LastJoinExitResp
	RateProviders                     []common.Address
	TokenRateCaches                   []TokenRateCacheResp
	SwapFeePercentage                 *big.Int
	ProtocolFeePercentageCache        map[int]*big.Int
	IsTokenExemptFromYieldProtocolFee []bool
	IsExemptFromYieldProtocolFee      bool
	InRecoveryMode                    bool
	PausedState                       PausedStateResp
	BlockNumber                       uint64
}
