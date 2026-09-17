package deepstateob

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

// poolID mirrors DeepstateV1.poolId(token0,token1): keccak256 of the two
// addresses, each left-padded to a 32-byte word (Solidity abi.encode, not
// abi.encodePacked).
func poolID(token0, token1 common.Address) [32]byte {
	buf := make([]byte, 0, 64)
	buf = append(buf, common.LeftPadBytes(token0.Bytes(), 32)...)
	buf = append(buf, common.LeftPadBytes(token1.Bytes(), 32)...)
	return [32]byte(crypto.Keccak256(buf))
}

// bookID mirrors DeepstateV1.bookId(token0,token1,epoch): keccak256 of the
// two addresses and the epoch, each left-padded to a 32-byte word.
func bookID(token0, token1 common.Address, epoch *big.Int) [32]byte {
	buf := make([]byte, 0, 96)
	buf = append(buf, common.LeftPadBytes(token0.Bytes(), 32)...)
	buf = append(buf, common.LeftPadBytes(token1.Bytes(), 32)...)
	buf = append(buf, common.LeftPadBytes(epoch.Bytes(), 32)...)
	return [32]byte(crypto.Keccak256(buf))
}
