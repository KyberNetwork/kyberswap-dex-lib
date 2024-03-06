package valueobject

type ChainID uint

const (
	ChainIDEthereum        ChainID = 1
	ChainIDEthw            ChainID = 10001
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
	ChainIDZKSync          ChainID = 324
	ChainIDLineaGoerli     ChainID = 59140
	ChainIDLinea           ChainID = 59144
	ChainIDPolygonZkEVM    ChainID = 1101
	ChainIDBase            ChainID = 8453
	ChainIDScroll          ChainID = 534352
	ChainIDBlast           ChainID = 81457
)

var l2EncoderSupportedChains = map[ChainID]struct{}{
	ChainIDArbitrumOne: {},
	ChainIDOptimism:    {},
	ChainIDBase:        {},
	ChainIDBlast:       {},
}

func IsL2EncoderSupportedChains(chainID ChainID) bool {
	_, exist := l2EncoderSupportedChains[chainID]
	return exist
}
