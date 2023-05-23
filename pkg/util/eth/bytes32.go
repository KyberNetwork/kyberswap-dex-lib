package eth

import "github.com/ethereum/go-ethereum/common"

func StringToBytes32(src string) (dest [32]byte) {
	copy(dest[:], common.FromHex(src))

	return
}

func Bytes32ToString(src [32]byte) (dest string) {
	return common.BytesToHash(src[:]).String()
}
