package token

import (
	"hash/fnv"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

func HashToken(address string) (uint32, error) {
	addr, err := hexutil.Decode(address)
	if err != nil {
		return 0, err
	}
	h := fnv.New32a()
	h.Write(addr)
	return h.Sum32(), nil
}
