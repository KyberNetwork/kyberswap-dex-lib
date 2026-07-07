package rsethl2

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

type Config struct {
	DexID             string              `json:"dexID"`
	LRTDepositPool    string              `json:"lrtDepositPool"`
	ChainId           valueobject.ChainID `json:"chainId"`
	CheckNative       bool                `json:"checkNative"`
	NoSupportedTokens bool                `json:"noSupportedTokens"`
}
