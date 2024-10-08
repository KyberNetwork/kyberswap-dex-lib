package generic_simple_rate

import (
	"fmt"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

var bytesByPathMap = map[string]map[string][]byte{}

var abiByPathMap = map[string]abi.ABI{}

func GetBytesByPath(dexID string, path string) ([]byte, error) {
	if bytesByPath, ok := bytesByPathMap[dexID]; ok {
		if data, ok := bytesByPath[path]; ok {
			return data, nil
		}
		return nil, fmt.Errorf("path not found: %s", path)
	}
	return nil, fmt.Errorf("unknown folder: %s%s", dexID, path)
}

func GetABI(exchange string) abi.ABI {
	if ABI, ok := abiByPathMap[exchange]; ok {
		return ABI
	}
	return rateABI
}
