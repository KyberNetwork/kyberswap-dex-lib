package constant

import (
	coreEntities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/ethereum/go-ethereum/common"
)

// Known WETH9 implementation addresses, used in our implementation of Ether#wrapped
var WETH9 = map[uint]*coreEntities.Token{
	1:     coreEntities.NewToken(1, common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"), 18, "WETH", "Wrapped Ether"),
	10001: coreEntities.NewToken(1, common.HexToAddress("0xC02aaA39b223FE8D0A0e5C4F27eAD9083C756Cc2"), 18, "WETH", "Wrapped Ether"),
	3:     coreEntities.NewToken(3, common.HexToAddress("0xc778417E063141139Fce010982780140Aa0cD5Ab"), 18, "WETH", "Wrapped Ether"),
	4:     coreEntities.NewToken(4, common.HexToAddress("0xc778417E063141139Fce010982780140Aa0cD5Ab"), 18, "WETH", "Wrapped Ether"),
	5:     coreEntities.NewToken(5, common.HexToAddress("0xB4FBF271143F4FBf7B91A5ded31805e42b2208d6"), 18, "WETH", "Wrapped Ether"),
	42:    coreEntities.NewToken(42, common.HexToAddress("0xd0A1E359811322d97991E03f863a0C30C2cF029C"), 18, "WETH", "Wrapped Ether"),

	56: coreEntities.NewToken(56, common.HexToAddress("0xbb4CdB9CBd36B01bD1cBaEBF2De08d9173bc095c"), 18, "WBNB", "Wrapped BNB"),

	10: coreEntities.NewToken(10, common.HexToAddress("0x4200000000000000000000000000000000000006"), 18, "WETH", "Wrapped Ether"),
	69: coreEntities.NewToken(69, common.HexToAddress("0x4200000000000000000000000000000000000006"), 18, "WETH", "Wrapped Ether"),

	137:        coreEntities.NewToken(137, common.HexToAddress("0x0d500B1d8E8eF31E21C99d1Db9A6444d3ADf1270"), 18, "WMATIC", "Wrapped Matic"),
	80001:      coreEntities.NewToken(80001, common.HexToAddress("0x19395624C030A11f58e820C3AeFb1f5960d9742a"), 18, "WMUMBAI", "Wrapped Mumbai"),
	43114:      coreEntities.NewToken(43114, common.HexToAddress("0xB31f66AA3C1e785363F0875A1B74E27b85FD66c7"), 18, "WAVAX", "Wrapped AVAX"),
	43113:      coreEntities.NewToken(43114, common.HexToAddress("0x1D308089a2D1Ced3f1Ce36B1FcaF815b07217be3"), 18, "WAVAX", "Wrapped AVAX"),
	250:        coreEntities.NewToken(250, common.HexToAddress("0x21be370D5312f44cB42ce377BC9b8a0cEF1A4C83"), 18, "WFTM", "Wrapped Fantom"),
	25:         coreEntities.NewToken(25, common.HexToAddress("0x5C7F8A570d578ED84E63fdFA7b1eE72dEae1AE23"), 18, "WCRO", "Wrapped CRO"),
	199:        coreEntities.NewToken(199, common.HexToAddress("0x8D193c6efa90BCFf940A98785d1Ce9D093d3DC8A"), 18, "WBTT", "Wrapped BitTorrent"),
	106:        coreEntities.NewToken(106, common.HexToAddress("0xc579D1f3CF86749E05CD06f7ADe17856c2CE3126"), 18, "WVLX", "Wrapped VLX"),
	1313161554: coreEntities.NewToken(1313161554, common.HexToAddress("0xC9BdeEd33CD01541e1eeD10f90519d2C06Fe3feB"), 18, "WROSE", "Wrapped Rose"),
	42262:      coreEntities.NewToken(42262, common.HexToAddress("0x21C718C22D52d0F3a789b752D4c2fD5908a8A733"), 18, "WETH", "Wrapped ETH"),
	42161:      coreEntities.NewToken(42161, common.HexToAddress("0x82aF49447D8a07e3bd95BD0d56f35241523fBab1"), 18, "WETH", "Wrapped Ether"),
	421611:     coreEntities.NewToken(421611, common.HexToAddress("0xB47e6A5f8b33b3F17603C83a0535A9dcD7E32681"), 18, "WETH", "Wrapped Ether"),
}
