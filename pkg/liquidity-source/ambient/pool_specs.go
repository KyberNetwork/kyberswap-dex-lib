package ambient

import (
	"errors"
	"fmt"
	"math/big"
)

/* @notice Given a mapping of pools, a base/quote token pair and a pool type index,
*         copies the pool specification to memory. */
func queryPool(tokenX string, tokenY string, poolIdx *big.Int) (*swapPool, error) {
	// bytes32 key = encodeKey(tokenX, tokenY, poolIdx);
	key, err := encodeKey(tokenX, tokenY, poolIdx)
	if err != nil {
		return nil, err
	}

	// Pool memory pool = pools[key];
	return pools(key), nil

	// WONTDO
	// address oracle = oracleForPool(poolIdx, pool.oracleFlags_);
	// return PoolCursor ({head_: pool, hash_: key, oracle_: oracle});
}

/* @notice Hashes the key associated with a pool for a base/quote asset pair and
*         a specific pool type index. */
func encodeKey(tokenX string, tokenY string, poolIdx *big.Int) (string, error) {
	// 	 require(tokenX < tokenY);
	if tokenX > tokenY {
		return "", errors.New("invalid token param")
	}

	// 	 return keccak256(abi.encode(tokenX, tokenY, poolIdx));
	return fmt.Sprintf("%s-%s-%s", tokenX, tokenY, poolIdx.String()), nil
}

func pools(_ string) *swapPool {
	return nil
}
