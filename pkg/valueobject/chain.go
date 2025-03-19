package valueobject

import (
	"strconv"
)

type ChainID uint

const (
	ChainIDEthereum        ChainID = 1
	ChainIDRopsten         ChainID = 3
	ChainIDRinkeBy         ChainID = 4
	ChainIDGoerli          ChainID = 5
	ChainIDOptimism        ChainID = 10
	ChainIDKovan           ChainID = 42
	ChainIDBSC             ChainID = 56
	ChainIDOptimismKovan   ChainID = 69
	ChainIDPolygon         ChainID = 137
	ChainIDMumbai          ChainID = 80001
	ChainIDAvalancheCChain ChainID = 43114
	ChainIDFantom          ChainID = 250
	ChainIDCronos          ChainID = 25
	ChainIDBitTorrent      ChainID = 199
	ChainIDVelasEVM        ChainID = 106
	ChainIDAurora          ChainID = 1313161554
	ChainIDOasisEmerald    ChainID = 42262
	ChainIDArbitrumOne     ChainID = 42161
	ChainIDArbitrumRinkeby ChainID = 421611
	ChainIDEthereumW       ChainID = 10001
	ChainIDFuji            ChainID = 43113
	ChainIDLineaGoerli     ChainID = 59140
	ChainIDLinea           ChainID = 59144
	ChainIDZKSync          ChainID = 324
	ChainIDPolygonZkEVM    ChainID = 1101
	ChainIDBase            ChainID = 8453
	ChainIDScroll          ChainID = 534352
	ChainIDBlast           ChainID = 81457
	ChainIDMantle          ChainID = 5000
	ChainIDSonic           ChainID = 146
	ChainIDBerachain       ChainID = 80094
	ChainIDRonin           ChainID = 2020

	// ChainIDSolana is currently used in case of store price to db, that we should transform token addr into lowercase or not.
	ChainIDSolana ChainID = 0
)

var ChainNameMap = map[ChainID]string{
	ChainIDEthereum:        "ethereum",
	ChainIDRopsten:         "ethereum-ropsten",
	ChainIDRinkeBy:         "ethereum-rinkeby",
	ChainIDGoerli:          "ethereum-goerli",
	ChainIDOptimism:        "optimism",
	ChainIDKovan:           "ethereum-kovan",
	ChainIDBSC:             "bsc",
	ChainIDOptimismKovan:   "optimism-kovan",
	ChainIDPolygon:         "polygon",
	ChainIDMumbai:          "mumbai",
	ChainIDAvalancheCChain: "avalanche",
	ChainIDFantom:          "fantom",
	ChainIDCronos:          "cronos",
	ChainIDBitTorrent:      "bttc",
	ChainIDVelasEVM:        "velas",
	ChainIDAurora:          "aurora",
	ChainIDOasisEmerald:    "oasis",
	ChainIDArbitrumOne:     "arbitrum",
	ChainIDArbitrumRinkeby: "arbitrum-rinkeby",
	ChainIDEthereumW:       "ethereum-w",
	ChainIDFuji:            "avalanche-fuji",
	ChainIDLineaGoerli:     "linea-goerli",
	ChainIDLinea:           "linea",
	ChainIDZKSync:          "zkSync",
	ChainIDPolygonZkEVM:    "polygon-zkEVM",
	ChainIDBase:            "base",
	ChainIDScroll:          "scroll",
	ChainIDBlast:           "blast",
	ChainIDMantle:          "mantle",
	ChainIDSonic:           "sonic",
	ChainIDBerachain:       "berachain",
	ChainIDRonin:           "ronin",

	ChainIDSolana: "solana",
}

func (c ChainID) String() string {
	str, err := ToString(c)
	if err != nil {
		return strconv.Itoa(int(c))
	}
	return str
}

func ToString(chainID ChainID) (string, error) {
	if name, ok := ChainNameMap[chainID]; ok {
		return name, nil
	} else {
		return "", ErrChainUnsupported
	}
}
