package uniswapv4

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"

	uniswapv3 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v3"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "uniswap-v4"

	graphFirstLimit = 1000

	maxChangedTicks = 10

	tickChunkSize = 100
)

var (
	PoolManager = map[valueobject.ChainID]common.Address{
		valueobject.ChainIDArbitrumOne:     common.HexToAddress("0x360e68faccca8ca495c1b759fd9eee466db9fb32"),
		valueobject.ChainIDAvalancheCChain: common.HexToAddress("0x06380c0e0912312b5150364b9dc4542ba0dbbc85"),
		valueobject.ChainIDBase:            common.HexToAddress("0x498581ff718922c3f8e6a244956af099b2652b2b"),
		valueobject.ChainIDBSC:             common.HexToAddress("0x28e2ea090877bf75740558f6bfb36a5ffee9e9df"),
		valueobject.ChainIDEthereum:        common.HexToAddress("0x000000000004444c5dc75cB358380D2e3dE08A90"),
		valueobject.ChainIDMonad:           common.HexToAddress("0x188d586ddcf52439676ca21a244753fa19f9ea8e"),
		valueobject.ChainIDMegaETH:         common.HexToAddress("0x58DD83c317B03e6eBD72C3e912adF60a8e97Aa95"),
		valueobject.ChainIDOptimism:        common.HexToAddress("0x9a13f98cb987694c9f086b1f5eb990eea8264ec3"),
		valueobject.ChainIDPolygon:         common.HexToAddress("0x67366782805870060151383f4bbff9dab53e5cd6"),
		valueobject.ChainIDUnichain:        common.HexToAddress("0x1f98400000000000000000000000000000000004"),
	}

	// NativeTokenAddress is the address that UniswapV4 uses to represent native token in pools.
	NativeTokenAddress = valueobject.AddrZero

	ErrTooManyChangedTicks = errors.New("too many changed ticks")

	ErrInvalidAmountIn  = errors.New("invalid amount in")
	ErrInvalidAmountOut = errors.New("invalid amount out")
	ErrInvalidFee       = errors.New("invalid fee")

	defaultGas = uniswapv3.Gas{BaseGas: 129869, CrossInitTickGas: 15460}
)
