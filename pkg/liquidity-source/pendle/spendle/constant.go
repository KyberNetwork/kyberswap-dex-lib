package spendle

import (
	"errors"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

const (
	DexType = "spendle"

	gasStake   = 330000
	gasUnstake = 330000
)

var (
	ErrInvalidAmount = errors.New("invalid amount")

	POne = bignumber.TenPowInt(18)
)
