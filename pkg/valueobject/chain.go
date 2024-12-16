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

	// ChainIDSolana is currently used in case of store price to db, that we should transform token addr into lowercase or not.
	ChainIDSolana ChainID = 0
)

func (c ChainID) String() string {
	str, err := ToString(c)
	if err != nil {
		return strconv.Itoa(int(c))
	}
	return str
}

func ToString(chainID ChainID) (string, error) {
	switch chainID {
	case 1:
		return "ethereum", nil
	case 3:
		return "ethereum-ropsten", nil
	case 4:
		return "ethereum-rinkeby", nil
	case 5:
		return "ethereum-goerli", nil
	case 10:
		return "optimism", nil
	case 42:
		return "ethereum-kovan", nil
	case 56:
		return "bsc", nil
	case 69:
		return "optimism-kovan", nil
	case 137:
		return "polygon", nil
	case 80001:
		return "mumbai", nil
	case 43114:
		return "avalanche", nil
	case 250:
		return "fantom", nil
	case 25:
		return "cronos", nil
	case 199:
		return "bttc", nil
	case 106:
		return "velas", nil
	case 1313161554:
		return "aurora", nil
	case 42262:
		return "oasis", nil
	case 42161:
		return "arbitrum", nil
	case 421611:
		return "arbitrum-rinkeby", nil
	case 10001:
		return "ethw", nil
	case 43113:
		return "fuji", nil
	case 59140:
		return "linea-goerli", nil
	case 59144:
		return "linea", nil
	case 324:
		return "zksync", nil
	case 1101:
		return "polygon-zkevm", nil
	case 8453:
		return "base", nil
	case 534352:
		return "scroll", nil
	case 81457:
		return "blast", nil
	case 5000:
		return "mantle", nil
	case 0:
		return "solana", nil
	case 146:
		return "sonic", nil
	default:
		return "", ErrChainUnsupported
	}

}
