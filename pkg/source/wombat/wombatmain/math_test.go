package wombatmain_test

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat/wombatmain"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQuotePotentialSwap(t *testing.T) {
	// https://etherscan.io/address/0x0020a8890e723cd94660a5404c4bccbb91680db6#readProxyContract
	fromToken := "0x1a7e4e63778B4f12a199C062f3eFdD288afCBce8"
	toToken := "0x3231Cb76718CDeF2155FC47b5286d82e6eDA273f"
	fromAmount := bignumber.NewBig10("1000000")
	haircutRate := bignumber.NewBig10("100000000000000")
	ampFactor := bignumber.NewBig10("2500000000000000")
	startCovRatio := bignumber.NewBig10("1500000000000000000")
	endCovRatio := bignumber.NewBig10("1800000000000000000")
	assetMap := map[string]wombat.Asset{
		"0x1a7e4e63778B4f12a199C062f3eFdD288afCBce8": {
			Cash:                    bignumber.NewBig10("46951671089794928947014"),
			Liability:               bignumber.NewBig10("65972348694976128221728"),
			UnderlyingTokenDecimals: 18,
		},
		"0x3231Cb76718CDeF2155FC47b5286d82e6eDA273f": {
			Cash:                    bignumber.NewBig10("92750509355503762003176"),
			Liability:               bignumber.NewBig10("73700859102292645892616"),
			UnderlyingTokenDecimals: 18,
		},
	}

	potentialOutcome, haircut, err := wombatmain.QuotePotentialSwap(
		fromToken, toToken, fromAmount,
		haircutRate, ampFactor, startCovRatio, endCovRatio, assetMap)

	assert.Nil(t, err)
	assert.Equal(t, "1031709", potentialOutcome.String())
	assert.Equal(t, "103", haircut.String())
}
