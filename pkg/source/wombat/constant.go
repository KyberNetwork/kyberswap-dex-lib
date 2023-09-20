package wombat

import "time"

const (
	DexTypeWombat = "wombat"

	poolTypeWombatLSD  = "wombat-lsd"
	poolTypeWombatMain = "wombat-main"

	assetMethodGetRelativePrice = "getRelativePrice"
	assetMethodCash             = "cash"
	assetMethodLiability        = "liability"

	poolMethodAddressOfAsset = "addressOfAsset"
	poolMethodAmpFactor      = "ampFactor"
	poolMethodEndCovRatio    = "endCovRatio"
	poolMethodHaircutRate    = "haircutRate"
	poolMethodStartCovRatio  = "startCovRatio"
	poolMethodIsPaused       = "isPaused"

	graphQLRequestTimeout = 20 * time.Second
)

var (
	defaultTokenWeight uint = 50
	zeroString              = "0"
)
