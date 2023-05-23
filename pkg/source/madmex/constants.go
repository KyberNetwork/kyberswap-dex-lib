package madmex

const DexTypeMadmex = "madmex"

const flagArbitrumSeqOffline = "0xa438451d6458044c3c8cd2f6f31c91ac882a6d91"

var secondaryPriceFeedVersionByChainID = map[ChainID]SecondaryPriceFeedVersion{
	POLYGON: SecondaryPriceFeedVersion1,
}
