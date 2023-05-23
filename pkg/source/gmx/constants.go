package gmx

const DexTypeGmx = "gmx"

const flagArbitrumSeqOffline = "0xa438451d6458044c3c8cd2f6f31c91ac882a6d91"

var secondaryPriceFeedVersionByChainID = map[ChainID]SecondaryPriceFeedVersion{
	ARBITRUM:  secondaryPriceFeedVersion2,
	AVALANCHE: secondaryPriceFeedVersion2,
}
