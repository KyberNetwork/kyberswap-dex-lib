package wombat

const (
	DexTypeWombat = "wombat"

	PoolTypeWombatLSD        = "wombat-lsd"
	PoolTypeWombatMain       = "wombat-main"
	PoolTypeWombatCrossChain = "wombat-cross-chain"

	assetMethodGetRelativePrice = "getRelativePrice"
	assetMethodCash             = "cash"
	assetMethodLiability        = "liability"

	poolMethodAddressOfAsset         = "addressOfAsset"
	poolMethodIsPaused               = "isPaused"
	poolMethodAmpFactor              = "ampFactor"
	poolMethodEndCovRatio            = "endCovRatio"
	poolMethodHaircutRate            = "haircutRate"
	poolMethodStartCovRatio          = "startCovRatio"
	poolMethodPaused                 = "paused"
	poolMethodCreditForTokensHaircut = "creditForTokensHaircut"
)

var (
	defaultTokenWeight uint = 50
	zeroString              = "0"
)
