package metavault

const DexTypeMetavault = "metavault"

type ChainID int

type SecondaryPriceFeedVersion int

const MATIC ChainID = 137

var FlagArbitrumSeqOffline = "0xa438451d6458044c3c8cd2f6f31c91ac882a6d91"

const (
	SecondaryPriceFeedVersion1       SecondaryPriceFeedVersion = 1
	SecondaryPriceFeedVersion2       SecondaryPriceFeedVersion = 2
	DefaultSecondaryPriceFeedVersion                           = SecondaryPriceFeedVersion2
)

var SecondaryPriceFeedVersionByChainID = map[ChainID]SecondaryPriceFeedVersion{
	MATIC: SecondaryPriceFeedVersion2,
}
