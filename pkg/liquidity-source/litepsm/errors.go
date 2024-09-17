package litepsm

import "errors"

var (
	ErrSellGemHalted          = errors.New("DssLitePsm/sell-gem-halted")
	ErrBuyGemHalted           = errors.New("DssLitePsm/buy-gem-halted")
	ErrOverflow               = errors.New("overflow")
	ErrInsufficientDAIBalance = errors.New("inssufficient dai balance")
	ErrInsufficientGemBalance = errors.New("inssufficient gem balance")
	ErrInvalidToken           = errors.New("invalid token")
)
