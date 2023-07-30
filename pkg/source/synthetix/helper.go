package synthetix

import "github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"

func getPoolStateVersion(chainID valueobject.ChainID) PoolStateVersion {
	poolStateVersion, ok := PoolStateVersionByChainID[chainID]
	if !ok {
		return DefaultPoolStateVersion
	}

	return poolStateVersion
}
