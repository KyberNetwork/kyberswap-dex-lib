package generic_simple_rate

import (
	"fmt"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/oeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/wbeth"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
	"github.com/ethereum/go-ethereum/accounts/abi"
)

var bytesByPathMap = map[string]map[string][]byte{
	string(valueobject.ExchangeWBETH): wbeth.BytesByPath,
	string(valueobject.ExchangeOETH):  oeth.BytesByPath,
}

var abiByPathMap = map[string]abi.ABI{
	string(valueobject.ExchangeWBETH): wbeth.WBETHABI,
}

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
