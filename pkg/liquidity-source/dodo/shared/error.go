package shared

import "errors"

var (
	ErrPoolAddressBanned         = errors.New("poolAddress was banned")
	ErrInitializeBlacklistFailed = errors.New("initialize DODO black list failed")
	ErrStaticExtraEmpty          = errors.New("staticExtra is empty")
	ErrExtraEmpty                = errors.New("extra is empty")
	ErrInvalidToken              = errors.New("invalid token")
)
