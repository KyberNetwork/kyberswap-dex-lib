package oneinchaqua

import (
	_ "embed"

	abiutil "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
)

//go:embed abi/Aqua.json
var aquaABIJson []byte

// The handles are looked up at startup so a renamed method or event fails at init, not in a refresh.
var (
	aquaABI  = abiutil.MustParseABI(aquaABIJson)
	erc20ABI = abiutil.Erc20ABI

	rawBalancesMethod = abiutil.MustABIMethod(aquaABI, "rawBalances")
	balanceOfMethod   = abiutil.MustABIMethod(erc20ABI, "balanceOf")
	allowanceMethod   = abiutil.MustABIMethod(erc20ABI, "allowance")

	shippedEvent = abiutil.MustABIEvent(aquaABI, "Shipped")
	pushedEvent  = abiutil.MustABIEvent(aquaABI, "Pushed")
	dockedEvent  = abiutil.MustABIEvent(aquaABI, "Docked")
)
