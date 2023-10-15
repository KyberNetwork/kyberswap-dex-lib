package gmxglp

import "math/big"

const (
	secondaryPriceFeedVersion1 SecondaryPriceFeedVersion = 1
	secondaryPriceFeedVersion2 SecondaryPriceFeedVersion = 2

	calcAmountOutTypeStake   = "stake"
	calcAmountOutTypeUnStake = "un-stake"
)

type VaultAddress struct {
	Vault string `json:"vault"`
}

type Extra struct {
	Vault           *Vault           `json:"vault"`
	GlpManager      *GlpManager      `json:"glpManager"`
	YearnTokenVault *YearnTokenVault `json:"yearnTokenVault"`
}

type Meta struct {
	StakeGLP      string `json:"stakeGLP"`
	GlpManager    string `json:"glpManager"`
	YearnVault    string `json:"yearnVault"`
	DirectionFlag uint8  `json:"directionFlag"`
}

type ChainID uint

type SecondaryPriceFeedVersion int

type gmxGlpSwapInfo struct {
	calcAmountOutType string
	mintAmount        *big.Int
	amountAfterFees   *big.Int
	redemptionAmount  *big.Int
	usdgAmount        *big.Int
}
