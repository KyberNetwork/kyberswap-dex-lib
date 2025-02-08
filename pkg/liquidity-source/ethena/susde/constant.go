package susde

import (
	"errors"
)

const (
	DexType = "ethena-susde"

	StakedUSDeV2 = "0x9d39a5de30e57443bff2a8307a4256c8797a3497"
	USDe         = "0x4c9edd5852cd905f086c759e8383e09bff1e68b3"

	stakedUSDeV2MethodAsset       = "asset"
	stakedUSDeV2MethodTotalSupply = "totalSupply"
	stakedUSDeV2MethodTotalAssets = "totalAssets"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrOverflow     = errors.New("overflow")
)

var (
	defaultGas = Gas{Deposit: 58500}
)
