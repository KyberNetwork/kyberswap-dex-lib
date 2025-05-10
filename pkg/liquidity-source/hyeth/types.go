package hyeth

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

type Extra struct {
	ManagerIssueFee           *uint256.Int   `json:"feeI"`
	ManagerRedeemFee          *uint256.Int   `json:"feeR"`
	Component                 common.Address `json:"comp"`
	ComponentTotalSupply      *uint256.Int   `json:"compSup"`
	ComponentTotalAsset       *uint256.Int   `json:"compAss"`
	ComponentHyethBalance     *uint256.Int   `json:"compHyb"`
	HyethTotalSupply          *uint256.Int   `json:"hySup"`
	DefaultPositionRealUnit   *uint256.Int   `json:"dpru"`
	ExternalPositionRealUnits []*uint256.Int `json:"epru"`
	IsDisabled                bool           `json:"isDisabled"`
	MaxDeposit                *uint256.Int   `json:"maxDeposit"`
	MaxRedeem                 *uint256.Int   `json:"maxRedeem"`
}

type PoolItem struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	LpToken string             `json:"lpToken"`
	Tokens  []entity.PoolToken `json:"tokens"`
}

type SwapInfo struct {
	Fee         *uint256.Int `json:"fee"`
	TotalSupply *uint256.Int `json:"totalSupply"`
	TotalAssets *uint256.Int `json:"totalAssets"`
}
