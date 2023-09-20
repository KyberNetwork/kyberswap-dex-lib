package wombatlsd_test

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/wombat/wombatlsd"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestQuotePotentialSwap(t *testing.T) {
	// https://etherscan.io/address/0x647cc8816c2d60a5ff4d1ffef27a5b3637d5ac81#readProxyContract
	fromToken := "0xA35b1B31Ce002FBF2058D22F30f95D405200A15b"
	toToken := "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"
	fromAmount := bignumber.NewBig10("1000000000")
	haircutRate := bignumber.NewBig10("100000000000000")
	ampFactor := bignumber.NewBig10("2000000000000000")
	startCovRatio := bignumber.NewBig10("1500000000000000000")
	endCovRatio := bignumber.NewBig10("1800000000000000000")
	assetMap := map[string]wombat.Asset{
		"0xA35b1B31Ce002FBF2058D22F30f95D405200A15b": {
			Cash:                    bignumber.NewBig10("547370669405545596073"),
			Liability:               bignumber.NewBig10("516213215951692583758"),
			UnderlyingTokenDecimals: 18,
			RelativePrice:           bignumber.NewBig10("1007193313818254424"),
		},
		"0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2": {
			Cash:                    bignumber.NewBig10("301255868324403411564"),
			Liability:               bignumber.NewBig10("332480258276764034667"),
			UnderlyingTokenDecimals: 18,
			RelativePrice:           bignumber.NewBig10("1000000000000000000"),
		},
	}

	potentialOutcome, haircut, err := wombatlsd.QuotePotentialSwap(
		fromToken, toToken, fromAmount,
		haircutRate, ampFactor, startCovRatio, endCovRatio, assetMap)

	assert.Nil(t, err)
	assert.Equal(t, "1006432688", potentialOutcome.String())
	assert.Equal(t, "100653", haircut.String())
}
