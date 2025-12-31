package ekubov3

import "errors"

const DexType = "ekubov3"

var (
	ErrZeroAmount = errors.New("zero amount")
	ErrReorg      = errors.New("reorg detected")
)
