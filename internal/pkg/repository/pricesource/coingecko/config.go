package coingecko

import "time"

// PlatformId can be fetched by using https://api.coingecko.com/api/v3/asset_platforms
// Remember to update this map when integrating new chain
var PlatformId = map[int]string{
	1:          "ethereum",
	137:        "polygon-pos",
	56:         "binance-smart-chain",
	43114:      "avalanche",
	250:        "fantom",
	25:         "cronos",
	42161:      "arbitrum-one",
	199:        "", // Coingecko does not support BTTC yet
	106:        "velas",
	1313161554: "aurora",
	42262:      "oasis",
	10:         "optimistic-ethereum",
}

const (
	APIEndpoint  = "https://api.coingecko.com/api/v3"
	TimeoutLong  = 30 * time.Second
	TimeoutShort = 5 * time.Second
)
