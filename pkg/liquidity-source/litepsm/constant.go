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

	// Hop gas incl. the executor per-edge overhead (15,674), measured on Ethereum on 2026-11-28 via Tenderly
	// traces. Gas does not depend on the swap size.
	//   - 0xf6e72db5454dd049d0788e411b06cfaf16853042 (direct DssLitePsm): buy 62,003, sell 74,818.
	//   - 0xa188eec8f81263234da3622a406892f3d630f98c (USDS wrapper): buy 176,903, sell 192,113. gasWrap is the
	//     wrapper's extra Vat join/exit on top of the direct PSM (mean of 114,900 and 117,295).
	// gasJoin and gasMint are not measured.
	gasBuy  = 62003
	gasSell = 74818
	gasJoin = 88785
	gasWrap = 116098
	gasMint = 60753
)

var (
	HALTED = big256.UMax
)
