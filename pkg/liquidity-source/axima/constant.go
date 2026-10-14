package axima

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType    = "axima"
	defaultGas = 175000

	// metricBaseGas is the swap gas of a metric-propamm pool before extensions. Mean of 27 Tenderly pool-call
	// frames (base+bsc, both directions, POD included) of frame gas - extensions*gasPerExtension.
	metricBaseGas = 159433
	// gasPerExtension is the swap gas of one metric-propamm extension. Mean of 23 Tenderly hook frames
	// (base+bsc: priceVelocity, stopLoss, third extension).
	gasPerExtension = 18650
)

var Q64BI = new(big.Int).Lsh(bignumber.One, 64)
