package litepsm

import (
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

const DexTypeLitePSM = "lite-psm"

const (
	litePSMMethodPsm     = "psm"
	litePSMMethodPocket  = "pocket"
	litePSMMethodGemJoin = "gemJoin"
	litePSMMethodGem     = "gem"
	litePSMMethodTIn     = "tin"
	litePSMMethodTOut    = "tout"

	genericMethodDai = "dai"

	Precision = 18

	gasBuy  = 49634
	gasSell = 53309
	gasJoin = 88785
	gasWrap = 28031
	gasMint = 60753
)

var (
	HALTED = big256.UMax
)
