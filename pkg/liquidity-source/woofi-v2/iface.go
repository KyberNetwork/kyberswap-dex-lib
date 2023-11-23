package woofiv2

import (
	"math/big"

	"github.com/holiman/uint256"
)

// IWooracleV2
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/interfaces/IWooracleV2.sol#L39
type IWooracleV2 interface {
	State(base string) State
	Decimals(base string) uint8
}

// AggregatorV3Interface
// https://github.com/woonetwork/WooPoolV2/blob/e4fc06d357e5f14421c798bf57a251f865b26578/contracts/interfaces/AggregatorV3Interface.sol#L4
type AggregatorV3Interface interface {
	GetLatestRoundData() (*uint256.Int, *big.Int, *uint256.Int, *uint256.Int, *uint256.Int)
}
