package sellonly

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
)

var (
	// SellOnly lets anyone sell currency0 into the pool.
	SellOnly = common.HexToAddress("0x56C6038207e87424d3A1DAb544526f75761428a0")
	// SellOnlyOne lets anyone sell currency1 into the pool.
	SellOnlyOne = common.HexToAddress("0xF833A6F720Fc355b39C4D8d2E5a61E01D4E928A0")

	ErrWrongDirection = errors.New("sell-only hook: the pool only accepts the other direction")
)
