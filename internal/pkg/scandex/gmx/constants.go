package gmx

import (
	"github.com/KyberNetwork/router-service/internal/pkg/constant"
)

var FlagArbitrumSeqOffline = "0xa438451d6458044c3c8cd2f6f31c91ac882a6d91"

var SecondaryPriceFeedVersionByChainID = map[int]int{
	constant.ARBITRUM:  2,
	constant.AVALANCHE: 2,
}

var DefaultSecondaryPriceFeedVersion = 2
