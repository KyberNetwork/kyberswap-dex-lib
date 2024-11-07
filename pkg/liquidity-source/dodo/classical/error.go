package classical

import "errors"

var (
	ErrReserveDepleted      = errors.New("reserve depleted")
	ErrOnlySupportBuyBase   = errors.New("only support buy base")
	ErrBaseBalanceNotEnough = errors.New("DODO_BASE_BALANCE_NOT_ENOUGH")
	ErrInvalidRStatus       = errors.New("INVALID_R_STATUS")
	ErrPaidAmountTooLarge   = errors.New("paid amount is larger than swapAmount")
	ErrTradeNotAllowed      = errors.New("TRADE_NOT_ALLOWED")
	ErrSellingNotAllowed    = errors.New("SELLING_NOT_ALLOWED")
	ErrBuyingNotAllowed     = errors.New("BUYING_NOT_ALLOWED")
)
