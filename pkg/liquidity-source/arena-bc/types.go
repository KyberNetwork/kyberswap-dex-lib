package arenabc

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type TokenParametersResult struct {
	CurveScaler           *big.Int
	A                     uint16
	B                     uint8
	LpDeployed            bool
	LpPercentage          uint8
	SalePercentage        uint8
	CreatorFeeBasisPoints uint8
	CreatorAddress        common.Address
	PairAddress           common.Address
	TokenContractAddress  common.Address
}

func (t *TokenParametersResult) ToTokenParameters() *TokenParameters {
	return &TokenParameters{
		CurveScaler:           uint256.MustFromBig(t.CurveScaler),
		A:                     t.A,
		B:                     t.B,
		LpDeployed:            t.LpDeployed,
		LpPercentage:          t.LpPercentage,
		SalePercentage:        t.SalePercentage,
		CreatorFeeBasisPoints: t.CreatorFeeBasisPoints,
		PairAddress:           t.PairAddress,
	}
}

type TokenParameters struct {
	CurveScaler           *uint256.Int   `json:"cS"`
	A                     uint16         `json:"a"`
	B                     uint8          `json:"b"`
	LpDeployed            bool           `json:"lD"`
	LpPercentage          uint8          `json:"lP"`
	SalePercentage        uint8          `json:"sP"`
	CreatorFeeBasisPoints uint8          `json:"cFBP"`
	PairAddress           common.Address `json:"pA"`
}

type FeeData struct {
	ProtocolFee    *uint256.Int
	CreatorFee     *uint256.Int
	ReferralFee    *uint256.Int
	TotalFeeAmount *uint256.Int
}

type Extra struct {
	IsPaused              bool             `json:"p"`
	CanDeployLp           bool             `json:"cD"`
	TokenParams           *TokenParameters `json:"tP"`
	TokenSupply           *uint256.Int     `json:"tS"`
	TokenBalance          *uint256.Int     `json:"tB"`
	MaxTokensForSale      *uint256.Int     `json:"mTFS"`
	ProtocolFeeBasisPoint uint8            `json:"pFBP"`
	ReferralFeeBasisPoint uint8            `json:"rFBP"`
	AllowedTokenSupply    *uint256.Int     `json:"aTS"`
}

type StaticExtra struct {
	ChainId      valueobject.ChainID `json:"cI"`
	TokenManager string              `json:"tM"`
	TokenId      *big.Int            `json:"tI"`
}

type MetaInfo struct {
	BlockNumber     uint64 `json:"blockNumber"`
	ApprovalAddress string `json:"approvalAddress,omitempty"`
}

type SwapInfo struct {
	TokenManager            string       `json:"tM"`
	IsBuy                   bool         `json:"iB"`
	TokenId                 *big.Int     `json:"tI"`
	SwapAmount              *uint256.Int `json:"sA,omitempty"`
	MinScaledTokenAmountOut uint64       `json:"miSA,omitempty"`
	MaxScaledTokenAmountOut uint64       `json:"maSA,omitempty"`

	fee              *uint256.Int
	remainingTokenIn *pool.TokenAmount
	totalSupply      *uint256.Int
	nativeBalance    *uint256.Int
}
