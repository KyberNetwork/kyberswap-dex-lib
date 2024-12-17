package valueobject

import (
	"strings"
)

var WETHByChainID = map[ChainID]string{
	ChainIDEthereum:        "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
	ChainIDEthereumW:       "0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2",
	ChainIDRopsten:         "0xc778417E063141139Fce010982780140Aa0cD5Ab",
	ChainIDRinkeBy:         "0xc778417E063141139Fce010982780140Aa0cD5Ab",
	ChainIDGoerli:          "0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6",
	ChainIDKovan:           "0xd0A1E359811322d97991E03f863a0C30C2cF029C",
	ChainIDBSC:             "0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c",
	ChainIDOptimism:        "0x4200000000000000000000000000000000000006",
	ChainIDOptimismKovan:   "0x4200000000000000000000000000000000000006",
	ChainIDPolygon:         "0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270",
	ChainIDMumbai:          "0x19395624C030A11f58e820C3AeFb1f5960d9742a",
	ChainIDAvalancheCChain: "0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7",
	ChainIDFantom:          "0x21be370D5312f44cB42ce377BC9b8a0cEF1A4C83",
	ChainIDCronos:          "0x5C7F8A570d578ED84E63fdFA7b1eE72dEae1AE23",
	ChainIDBitTorrent:      "0x8D193c6efa90BCFf940A98785d1Ce9D093d3DC8A",
	ChainIDVelasEVM:        "0xc579D1f3CF86749E05CD06f7ADe17856c2CE3126",
	ChainIDAurora:          "0xC9BdeEd33CD01541e1eeD10f90519d2C06Fe3feB",
	ChainIDOasisEmerald:    "0x21C718C22D52d0F3a789b752D4c2fD5908a8A733",
	ChainIDArbitrumOne:     "0x82aF49447D8a07e3bd95BD0d56f35241523fBab1",
	ChainIDArbitrumRinkeby: "0xB47e6A5f8b33b3F17603C83a0535A9dcD7E32681",
	ChainIDLinea:           "0xe5D7C2a44FfDDf6b295A15c148167daaAf5Cf34f",
	ChainIDZKSync:          "0x5AEa5775959fBC2557Cc8789bC1bf90A239D9a91",
	ChainIDLineaGoerli:     "0x2c1b868d6596a18e32e61b901e4060c872647b6c",
	ChainIDPolygonZkEVM:    "0x4f9a0e7fd2bf6067db6994cf12e4495df938e6e9",
	ChainIDBase:            "0x4200000000000000000000000000000000000006",
	ChainIDScroll:          "0x5300000000000000000000000000000000000004",
	ChainIDBlast:           "0x4300000000000000000000000000000000000004",
	ChainIDMantle:          "0x78c1b0C915c4FAA5FffA6CAbf0219DA63d7f4cb8",
	ChainIDSonic:           "0x309C92261178fA0CF748A855e90Ae73FDb79EBc7",
}

// WrapETHLower wraps, if applicable, native token to wrapped token; and then lowercase it.
func WrapETHLower(token string, chainID ChainID) string {
	if strings.EqualFold(token, EtherAddress) {
		token = WETHByChainID[chainID]
	}
	return strings.ToLower(token)
}

func IsWETH(address string, chainID ChainID) bool {
	return strings.EqualFold(address, WETHByChainID[chainID])
}
