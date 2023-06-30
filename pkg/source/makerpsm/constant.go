package makerpsm

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"

const DexTypeMakerPSM = "maker-psm"

const (
	psmMethodTIn  = "tin"
	psmMethodTOut = "tout"
	psmMethodVat  = "vat"
	psmMethodIlk  = "ilk"
)

const (
	vatMethodIlks = "ilks"
	vatMethodDebt = "debt"
	vatMethodLine = "Line"
)

const (
	DAIAddress = "0x6b175474e89094c44da98b954eedeac495271d0f"
)

var (
	DefaultGas = Gas{SellGem: 115000, BuyGem: 115000}
	WAD        = bignumber.BONE
)
