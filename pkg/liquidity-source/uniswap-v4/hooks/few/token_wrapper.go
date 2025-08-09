package few

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap-v4/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type tokenWrapper struct{}

func NewTokenWrapper() tokenWrapper {
	return tokenWrapper{}
}

var canWrapToFew = map[valueobject.ChainID]map[string]TokenInfo{}
var isFewToken = map[valueobject.ChainID]map[string]TokenInfo{}

func (tokenWrapper) CanWrap(chainID valueobject.ChainID, address string) (shared.IWrapMetadata, bool) {
	value, ok := canWrapToFew[chainID][address]
	return value, ok
}

func (tokenWrapper) IsWrapped(chainID valueobject.ChainID, address string) (shared.IWrapMetadata, bool) {
	value, ok := isFewToken[chainID][address]
	return value, ok
}

func init() {
	for _, token := range fewTokens {
		if _, ok := canWrapToFew[token.ChainID]; !ok {
			canWrapToFew[token.ChainID] = make(map[string]TokenInfo)
		}
		canWrapToFew[token.ChainID][token.UnwrapTokenAddress] = token

		if _, ok := isFewToken[token.ChainID]; !ok {
			isFewToken[token.ChainID] = make(map[string]TokenInfo)
		}
		isFewToken[token.ChainID][token.FewTokenAddress] = token
	}
}
