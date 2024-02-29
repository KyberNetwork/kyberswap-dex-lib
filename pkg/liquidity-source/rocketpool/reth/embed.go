package reth

import _ "embed"

//go:embed abis/RocketDAOProtocolSettingsDeposit.json
var rocketDAOProtocolSettingsDepositABIJson []byte

//go:embed abis/RocketDepositPool.json
var rocketDepositPoolABIJSON []byte

//go:embed abis/RocketMinipoolQueue.json
var rocketMinipoolQueueABIJSON []byte

//go:embed abis/RocketNetworkBalances.json
var rocketNetworkBalancesABIJSON []byte

//go:embed abis/RocketTokenRETH.json
var rocketTokenRETHABIJSON []byte

//go:embed abis/RocketVault.json
var rocketVaultABIJSON []byte
