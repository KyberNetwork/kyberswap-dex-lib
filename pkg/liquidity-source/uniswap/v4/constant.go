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
		valueobject.ChainIDArbitrumOne:     common.HexToAddress("0x360E68faCcca8cA495c1B759Fd9EEe466db9FB32"),
		valueobject.ChainIDAvalancheCChain: common.HexToAddress("0x06380C0e0912312B5150364B9DC4542BA0DbBc85"),
		valueobject.ChainIDBase:            common.HexToAddress("0x498581fF718922c3f8e6A244956aF099B2652b2b"),
		valueobject.ChainIDBSC:             common.HexToAddress("0x28e2Ea090877bF75740558f6BFB36A5ffeE9e9dF"),
		valueobject.ChainIDEthereum:        common.HexToAddress("0x000000000004444c5dc75cB358380D2e3dE08A90"),
		valueobject.ChainIDMonad:           common.HexToAddress("0x188d586Ddcf52439676Ca21A244753fA19F9Ea8e"),
		valueobject.ChainIDMegaETH:         common.HexToAddress("0x58DD83c317B03e6eBD72C3e912adF60a8e97Aa95"),
		valueobject.ChainIDOptimism:        common.HexToAddress("0x9a13F98Cb987694C9F086b1F5eB990EeA8264Ec3"),
		valueobject.ChainIDPolygon:         common.HexToAddress("0x67366782805870060151383F4BbFF9daB53e5cD6"),
		valueobject.ChainIDUnichain:        common.HexToAddress("0x1F98400000000000000000000000000000000004"),
	}

	// Quoter maps chainID to the deployed Uniswap V4 Quoter contract address.
	// Used as fallback when HookQuoter is not available for a chain.
	// Addresses: https://docs.uniswap.org/contracts/v4/deployments
	Quoter = map[valueobject.ChainID]string{
		valueobject.ChainIDArbitrumOne:     "0x3972C00F7ED34D3ba0aF431F0D790be4E8B1e6E8",
		valueobject.ChainIDAvalancheCChain: "0x9F75dD27D6664c475B90e105573E550ff69437B0",
		valueobject.ChainIDBase:            "0x0d5e0F971ED27FBfF6c2837bf31316121532048D",
		valueobject.ChainIDBSC:             "0x9F75dD27D6664c475B90e105573E550ff69437B0",
		valueobject.ChainIDEthereum:        "0x52F0E24D1c21C8A0cB1e5a5dD6198556BD9E1203",
		valueobject.ChainIDMonad:           "0xa222Dd357A9076d1091Ed6Aa2e16C9742dD26891",
		valueobject.ChainIDMegaETH:         "0x94bDC671f0c35F44a1DaA53143fd1f868D1623b9",
		valueobject.ChainIDOptimism:        "0x9F75dD27D6664c475B90e105573E550ff69437B0",
		valueobject.ChainIDPolygon:         "0x9F75dD27D6664c475B90e105573E550ff69437B0",
		valueobject.ChainIDUnichain:        "0x333bA4e6c6c1Ae8E5BBa3a2c9C77d7c4Ea92f74b",
	}

	// NativeTokenAddress is the address that UniswapV4 uses to represent native token in pools.
	NativeTokenAddress = valueobject.AddrZero

	ErrTooManyChangedTicks = errors.New("too many changed ticks")
	ErrEmptyExtra          = errors.New("empty extra")

	ErrInvalidAmountIn     = errors.New("invalid amount in")
	ErrInvalidAmountOut    = errors.New("invalid amount out")
	ErrInvalidFee          = errors.New("invalid fee")
	ErrNilBeforeSwapResult = errors.New("before swap result is nil")
	ErrNilDeltaSpecified   = errors.New("delta specified is nil")
	ErrNilDeltaUnspecified = errors.New("delta unspecified is nil")

	defaultGas = uniswapv3.Gas{BaseGas: 129869, CrossInitTickGas: 15460}
)
