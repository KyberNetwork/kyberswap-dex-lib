package ekubo

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/ekubo/quoting/pool"
	"github.com/ethereum/go-ethereum/common"
)

type Config struct {
	Core        string
	DataFetcher string
	Router      string

	Extensions map[common.Address]pool.Extension

	ApiUrl string
}

var SepoliaConfig = Config{
	Core:        "0x16e186ecdc94083fff53ef2a41d46b92a54f61e2",
	DataFetcher: "0xe339a5e10f48d5c34255fd417f329d2026634b32",
	Router:      "0xab090b2d86a32ab9ed214224f59dc7453be1037e",

	Extensions: map[common.Address]pool.Extension{
		common.HexToAddress("0x51f1b10abf90e16498d25086641b0669ec62f32f"): pool.Oracle,
	},

	ApiUrl: "https://eth-sepolia-api.ekubo.org",
}
