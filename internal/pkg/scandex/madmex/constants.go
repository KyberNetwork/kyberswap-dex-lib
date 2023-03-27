package madmex

import (
	"github.com/KyberNetwork/kyberswap-aggregator/internal/pkg/constant"
)

var FlagArbitrumSeqOffline = "0xa438451d6458044c3c8cd2f6f31c91ac882a6d91"

var SecondaryPriceFeedVersionByChainID = map[int]int{
	constant.MATIC: 1,
}

var DefaultSecondaryPriceFeedVersion = 2
