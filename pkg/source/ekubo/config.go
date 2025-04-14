package ekubo

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting/pool"
	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	Core        string
	Oracle      string
	DataFetcher string
	Router      string

	Extensions map[common.Address]pool.Extension

	ApiUrl string
}

var MainnetConfig = Config{
	Core:        "0xe0e0e08A6A4b9Dc7bD67BCB7aadE5cF48157d444",
	Oracle:      "0x51d02a5948496a67827242eabc5725531342527c",
	DataFetcher: "0x91cB8a896cAF5e60b1F7C4818730543f849B408c",
	Router:      "0x9995855C00494d039aB6792f18e368e530DFf931",

	Extensions: map[common.Address]pool.Extension{
		common.HexToAddress("0x51d02a5948496a67827242eabc5725531342527c"): pool.Oracle,
	},

	ApiUrl: "https://eth-mainnet-api.ekubo.org",
}
