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
