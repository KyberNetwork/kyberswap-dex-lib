package brownfiv2

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"github.com/pkg/errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = "brownfi-v2"

	factoryMethodGetPair        = "allPairs"
	factoryMethodAllPairsLength = "allPairsLength"
	factoryMethodPriceFeedIds   = "priceFeedIds"

	pairMethodToken0      = "token0"
	pairMethodToken1      = "token1"
	pairMethodGetReserves = "getReserves"
	pairMethodFee         = "fee"
	pairMethodKappa       = "k"
	pairMethodLambda      = "lambda"

	pythDefaultBaseUrl         = "https://hermes.pyth.network"
	pythPathUpdatesPriceLatest = "v2/updates/price/latest"

	parsedDecimals = 18

	defaultGas = 387186
)

var (
	Router = map[valueobject.ChainID]common.Address{
		valueobject.ChainIDArbitrumOne: common.HexToAddress("0x3240853b71c89209ea8764CDDfA3b81766553E55"),
		valueobject.ChainIDBase:        common.HexToAddress("0x3240853b71c89209ea8764CDDfA3b81766553E55"),
		valueobject.ChainIDBerachain:   common.HexToAddress("0x3F0bBeEdEa5E5F63a14cBdA82718d4f25501fBeA"),
		valueobject.ChainIDBSC:         common.HexToAddress("0xD3F729D909a7E84669A35c3F25b37b4AC3487784"),
		valueobject.ChainIDHyperEVM:    common.HexToAddress("0x0A461D280891167Ee8391f4F0c03EECaa39ae632"),
	}

	q64       = big256.TwoPow64
	q64x2     = new(uint256.Int).Mul(q64, big256.U2)
	q128      = big256.TwoPow128
	precision = big256.TenPow(8)

	ErrFailToFetchPriceFeeds = errors.New("fail to fetch price feeds")

	ErrInvalidToken             = errors.New("invalid token")
	ErrInvalidReserve           = errors.New("invalid reserve")
	ErrInvalidPrices            = errors.WithMessage(pool.ErrUnsupported, "invalid prices")
	ErrInvalidAmountIn          = errors.New("invalid amount in")
	ErrInvalidAmountOut         = errors.New("invalid amount out")
	ErrInsufficientInputAmount  = errors.New("INSUFFICIENT_INPUT_AMOUNT")
	ErrInsufficientOutputAmount = errors.New("INSUFFICIENT_OUTPUT_AMOUNT")
	ErrMax80PercentOfReserve    = errors.New("MAX_80_PERCENT_OF_RESERVE")
)
