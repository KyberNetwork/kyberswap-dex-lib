package wombat

import (
	"github.com/holiman/uint256"
)

type Asset struct {
	IsPause                 bool         `json:"isPause"`
	Address                 string       `json:"address"`
	Cash                    *uint256.Int `json:"cash"`
	Liability               *uint256.Int `json:"liability"`
	UnderlyingTokenDecimals uint8        `json:"underlyingTokenDecimals"`
	RelativePrice           *uint256.Int `json:"relativePrice"`
}
