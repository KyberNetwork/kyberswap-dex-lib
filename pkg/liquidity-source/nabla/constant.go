package nabla

import (
	"errors"

	"github.com/KyberNetwork/int256"
	"github.com/ethereum/go-ethereum/common"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

const (
	DexType = valueobject.ExchangeNabla

	decimals = 18
)

var (
	priceScalingFactor = int256.NewInt(1e8)
	pricePrecision     = int256.NewInt(1e8)
	feePrecision       = int256.NewInt(1e6)

	mantissa = int256.NewInt(1e18)

	i1990 = int256.NewInt(1990)
	i1e3  = int256.NewInt(1000)
	i1e4  = int256.NewInt(10000)
	i1e6  = int256.NewInt(1000000)

	slot0 = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000000")
	slot1 = common.HexToHash("0x0000000000000000000000000000000000000000000000000000000000000001")
)

var (
	priceFeedIdToAsset = map[common.Hash]common.Address{
		common.HexToHash("0x962088abcfdbdb6e30db2e340c8cf887d9efb311b1f2f17b155a63dbb6d40265"): common.HexToAddress("0x6969696969696969696969696969696969696969"), // BERA/USD
		common.HexToHash("0xff61491a931112ddf1bd8147cd1b641375f79f5825126d665480874634fd0ace"): common.HexToAddress("0x2F6F07CDcf3588944Bf4C42aC74ff24bF56e7590"), // WETH/USD
		common.HexToHash("0xe62df6c8b4a85fe1a67db44dc12de5db330f7ac66b72dc658afedf0f4a415b43"): common.HexToAddress("0x0555E30da8f98308EdB960aa94C0Db47230d2B9c"), // WBTC/USD
		common.HexToHash("0xeaa020c61cc479712813461ce153894a96a6c00b21ed0cfc2798d1f9a9e9c94a"): common.HexToAddress("0x549943e04f40284185054145c6E4e9568C1D3241"), // USDC/USD
	}
)

var (
	ErrInvalidToken         = errors.New("invalid token")
	ErrInsufficientReserves = errors.New("insufficient reserves")
	ErrZeroSwap             = errors.New("zero swap")
)
