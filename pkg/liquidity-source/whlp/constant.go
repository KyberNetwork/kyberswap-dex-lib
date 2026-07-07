package whlp

import "github.com/ethereum/go-ethereum/common"

const (
	DexType          = "whlp"
	unlimitedReserve = "10000000000000000000000000"
)

var (
	accountantAddress = common.HexToAddress("0x470bd109A24f608590d85fc1f5a4B6e625E8bDfF")
	depositorAddress  = common.HexToAddress("0x340C9f6159ABc2bdfCC0E2b9Fe91D739006b41c1")
	usdt0Address      = common.HexToAddress("0xB8CE59FC3717ada4C02eaDF9682A9e934F625ebb")

	defaultGas int64 = 250000
)
