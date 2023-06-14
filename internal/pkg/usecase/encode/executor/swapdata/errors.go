package swapdata

import (
	"errors"
)

var (
	ErrMarshalFailed         = errors.New("marshal failed")
	ErrUnmarshalFailed       = errors.New("unmarshal failed")
	ErrSyncSwapVaultNotFound = errors.New("syncswap: vault is not found")
)
