package maplesyrup

import "errors"

const (
	DexType = "maple-syrup"

	poolManagerMethodActive       = "active"
	poolManagerMethodLiquidityCap = "liquidityCap"
)

var (
	ErrNotActive       = errors.New("P:NOT_ACTIVE")
	ErrDepositGtLiqCap = errors.New("P:DEPOSIT_GT_LIQ_CAP")
)
