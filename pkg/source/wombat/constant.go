package wombat

import "time"

const (
	DexTypeWombat = "wombat"

	PoolTypeWombatLSD        = "wombat-lsd"
	PoolTypeWombatMain       = "wombat-main"
	PoolTypeWombatCrossChain = "wombat-cross-chain"

	assetMethodGetRelativePrice = "getRelativePrice"
	assetMethodCash             = "cash"
	assetMethodLiability        = "liability"

	poolMethodAddressOfAsset         = "addressOfAsset"
	poolMethodAmpFactor              = "ampFactor"
	poolMethodEndCovRatio            = "endCovRatio"
	poolMethodHaircutRate            = "haircutRate"
	poolMethodStartCovRatio          = "startCovRatio"
	poolMethodPaused                 = "paused"
	poolMethodCreditForTokensHaircut = "creditForTokensHaircut"

	graphQLRequestTimeout = 20 * time.Second
)

var (
	defaultTokenWeight uint = 50
	zeroString              = "0"
)
