package generic_rate

import (
	"fmt"

	skypsm "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/maker/sky-psm"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

var bytesByPathMap = map[valueobject.Exchange]map[string][]byte{
	valueobject.ExchangeSkyPSM: skypsm.BytesByPath,
}

func GetBytesByPath(dexID string, path string) ([]byte, error) {
	if bytesByPath, ok := bytesByPathMap[valueobject.Exchange(dexID)][path]; ok {
		return bytesByPath, nil
	}
	return nil, fmt.Errorf("unknown folder: %s%s", dexID, path)
}
