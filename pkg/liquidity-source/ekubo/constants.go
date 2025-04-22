package ekubo

import "errors"

const DexType = "ekubo"

var (
	ErrZeroAmount = errors.New("zero amount")
	ErrReorg      = errors.New("reorg detected")
)
