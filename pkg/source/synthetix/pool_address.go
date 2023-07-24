package synthetix

import (
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// =============================================================================================
// Implementation of this contract:
// https://github.com/Uniswap/v3-periphery/blob/5bcdd9f67f9394f3159dad80d0dd01d37ca08c66/contracts/libraries/PoolAddress.sol

type PoolKey struct {
	token0 common.Address
	token1 common.Address
	fee    *big.Int
}

// @notice Returns PoolKey: the ordered tokens with the matched fee levels
// @param tokenA The first token of a pool, unsorted
// @param tokenB The second token of a pool, unsorted
// @param fee The fee level of the pool
// @return Poolkey The pool details with ordered token0 and token1 assignments
func getPoolKey(
	tokenA common.Address,
	tokenB common.Address,
	fee *big.Int,
) PoolKey {
	if tokenA.String() > tokenB.String() {
		tokenA, tokenB = tokenB, tokenA
	}

	return PoolKey{
		token0: tokenA,
		token1: tokenB,
		fee:    fee,
	}
}

// / @notice Deterministically computes the pool address given the factory and PoolKey
// / @param factoryAddress The Uniswap V3 factory contract address
// / @param key The PoolKey
// / @return pool The contract address of the V3 pool
func computeAddress(factoryAddress common.Address, key PoolKey) (common.Address, error) {
	if key.token0.String() >= key.token1.String() {
		return common.Address{}, ErrNotSortedKeys
	}

	var salt [32]byte
	addressTy, _ := abi.NewType("address", "address", nil)
	uint256Ty, _ := abi.NewType("uint256", "uint256", nil)

	arguments := abi.Arguments{{Type: addressTy}, {Type: addressTy}, {Type: uint256Ty}}

	bytes, _ := arguments.Pack(
		key.token0,
		key.token1,
		key.fee,
	)
	copy(salt[:], crypto.Keccak256(bytes))

	return crypto.CreateAddress2(factoryAddress, salt, common.FromHex(PoolInitCodeHash)), nil
}
