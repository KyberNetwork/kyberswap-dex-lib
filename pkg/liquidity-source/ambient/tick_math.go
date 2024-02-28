package ambient

import "math/big"

var (
	bigMaxSQRTRatio, _ = new(big.Int).SetString("21267430153580247136652501917186561138", 10)
	bigMinSQRTRatio    = big.NewInt(65538)
)
