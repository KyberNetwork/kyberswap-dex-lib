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
	// poc     1      1    0    0
	// gJ      1      0    1    1
	// dai     1      0    0    0
	// mint    0      0    1    1
	// gasBuy  = 48763 // 159435 44535 194841 203284
	// gasSell = 64670 // 171879 54584 195058 198522
	// 47316 + 88154.75g + 27942.75d + 60212m + 4487t
	gasBuy  = 49634
	gasSell = 53309
	gasJoin = 88785
	gasWrap = 28031
	gasMint = 60753
)

var (
	HALTED = big256.UMax
)
