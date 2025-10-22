package angstrom

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

const HASH_TO_STORE_KEY_SHIFT = 40

var mask24Bytes = big.NewInt(0xFFFFFF)
var bytes27Padding = make([]byte, 5)
var StorageSlotUnlockedFeesVariable = common.HexToHash("0x2") // _unlockedFees mapping

func keyFromAssetsUnchecked(asset0, asset1 common.Address) []byte {
	hash := crypto.Keccak256Hash(
		common.LeftPadBytes(asset0[:], 32),
		common.LeftPadBytes(asset1[:], 32),
	)

	bi := new(big.Int).SetBytes(hash[:])
	bi.Lsh(bi, HASH_TO_STORE_KEY_SHIFT)

	return bi.Bytes()[5:32]
}

func calculateUnlockedFeeSlot(key []byte, baseSlot common.Hash) common.Hash {
	return crypto.Keccak256Hash(key[:], bytes27Padding, baseSlot[:])
}

func extractUnlockedFee(data *big.Int) (unlockedFee *big.Int, protocolUnlockedFee *big.Int) {
	unlockedFee = new(big.Int).And(data, mask24Bytes)
	protocolUnlockedFee = new(big.Int).Rsh(data, 24)
	protocolUnlockedFee.And(protocolUnlockedFee, mask24Bytes)

	return
}
