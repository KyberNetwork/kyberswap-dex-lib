package midas

import (
	_ "embed"
)

//go:embed abi/DataFeed.json
var dataFeedBytes []byte

//go:embed abi/DepositVault.json
var depositVaultBytes []byte

//go:embed abi/RedemptionVault.json
var redemptionVaultBytes []byte

//go:embed abi/Redemption.json
var redemptionBytes []byte

//go:embed abi/RedemptionVaultWithUstb.json
var redemptionVaultWithUstbBytes []byte

//go:embed network/ethereum.json
var ethereumConfig []byte

var bytesByPath = map[string][]byte{
	"network/ethereum.json": ethereumConfig,
}
