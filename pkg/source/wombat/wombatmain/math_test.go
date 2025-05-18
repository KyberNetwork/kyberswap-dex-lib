package wombatmain

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

func TestQuotePotentialSwap(t *testing.T) {
	t.Parallel()
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

	potentialOutcome, haircut, _, _, err := Swap(
		fromToken, toToken, fromAmount, false,
		haircutRate, ampFactor, startCovRatio, endCovRatio, assetMap)

	assert.Nil(t, err)
	assert.Equal(t, "1031709", potentialOutcome.String())
	assert.Equal(t, "103", haircut.String())
}

func TestCovRatioLimitExceeded(t *testing.T) {
	t.Parallel()
	// https://bscscan.com/address/0x0520451b19ad0bb00ed35ef391086a692cfc74b2#readProxyContract
	fromToken := "0x55d398326f99059ff775485246999027b3197955"
	toToken := "0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d"
	fromAmount := bignumber.NewBig10("1509073456125711060849406")
	haircutRate := bignumber.NewBig10("20000000000000")
	ampFactor := bignumber.NewBig10("2500000000000000")
	startCovRatio := bignumber.NewBig10("1500000000000000000")
	endCovRatio := bignumber.NewBig10("1800000000000000000")
	assetMap := map[string]wombat.Asset{
		"0x55d398326f99059ff775485246999027b3197955": {
			Cash:                    bignumber.NewBig10("1754456575456286214822720"),
			Liability:               bignumber.NewBig10("1784169813200583235538664"),
			UnderlyingTokenDecimals: 18,
		},
		"0x8ac76a51cc950d9822d68b83fe1ad97b32cd580d": {
			Cash:                    bignumber.NewBig10("2135834898723293733588049"),
			Liability:               bignumber.NewBig10("1622189906391622039072609"),
			UnderlyingTokenDecimals: 18,
		},
	}

	_, _, _, _, err := Swap(
		fromToken, toToken, fromAmount, false,
		haircutRate, ampFactor, startCovRatio, endCovRatio, assetMap)

	assert.Equal(t, ErrCovRatioLimitExceeded, err)
}
