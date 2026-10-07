package navjit

import _ "embed"

var (
	//go:embed abis/NavJitHook.json
	hookABIJson []byte
	//go:embed abis/NavGuard.json
	navGuardABIJson []byte
	//go:embed abis/ERC20.json
	erc20ABIJson []byte
	//go:embed abis/V4Quoter.json
	quoterABIJson []byte
)
