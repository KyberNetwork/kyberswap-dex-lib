package dodo

import "errors"

var (
	ErrPoolAddressBanned         = errors.New("poolAddress was banned")
	ErrInitializeBlacklistFailed = errors.New("initialize DODO black list failed")
)
