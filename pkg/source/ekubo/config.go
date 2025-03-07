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

var SepoliaConfig = Config{
	Core:        "0xb98c50b291a8b69cffabd172de4d1bbc773f962a",
	Oracle:      "0x519c98252304a4933cdef1e66f139dfb0e2d2462",
	DataFetcher: "0x3b2b03c96f55c1a09a65d9c7e0b0abfe1816b02c",
	Router:      "0x82d25d06a00f04bae3a19107a8131afc019f3adf",

	Extensions: map[common.Address]pool.Extension{
		common.HexToAddress("0x519c98252304a4933cdef1e66f139dfb0e2d2462"): pool.Oracle,
	},

	ApiUrl: "https://eth-sepolia-api.ekubo.org",
}
