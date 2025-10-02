package arberaden

import (
	"errors"

	"github.com/holiman/uint256"
)

const (
	DexType = "arbera-den"
)

var (
	uONE = uint256.NewInt(1)
	u98  = uint256.NewInt(98)
	u100 = uint256.NewInt(100)
	DEN  = uint256.NewInt(1e4)
	Q96  = new(uint256.Int).Lsh(uONE, 96) // 1 << 96

	ErrInvalidToken  = errors.New("invalid token")
	ErrTokenNotExist = errors.New("token does not exist in pool assets")
)
