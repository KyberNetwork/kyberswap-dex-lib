package getcustomroute

import (
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

// collectTokenAddresses extracts addresses of pool tokens, combines with addresses and returns
func collectTokenAddresses(poolSet map[string]poolpkg.IPoolSimulator, addresses ...string) []string {
	tokenAddressSet := make(map[string]struct{}, len(poolSet)+len(addresses))
	for _, pool := range poolSet {
		for _, token := range pool.GetTokens() {
			tokenAddressSet[token] = struct{}{}
		}
	}

	for _, address := range addresses {
		tokenAddressSet[address] = struct{}{}
	}

	tokenAddresses := make([]string, 0, len(tokenAddressSet))
	for tokenAddress := range tokenAddressSet {
		tokenAddresses = append(tokenAddresses, tokenAddress)
	}

	return tokenAddresses
}
