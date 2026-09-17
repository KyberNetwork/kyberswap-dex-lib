package few

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

// TokenWrapper implements ITokenWrapper against a fixed list of TokenInfo.
// Each generation package (few/v1, few/v2, ...) constructs one of these from
// its own token list via NewTokenWrapper.
type TokenWrapper struct {
	canWrapToFew map[valueobject.ChainID]map[string]TokenInfo
	isFewToken   map[valueobject.ChainID]map[string]TokenInfo
}

// NewTokenWrapper builds a TokenWrapper indexing tokens by unwrap address
// (CanWrap) and by FewToken address (IsWrapped).
func NewTokenWrapper(tokens []TokenInfo) TokenWrapper {
	w := TokenWrapper{
		canWrapToFew: map[valueobject.ChainID]map[string]TokenInfo{},
		isFewToken:   map[valueobject.ChainID]map[string]TokenInfo{},
	}

	for _, token := range tokens {
		if _, ok := w.canWrapToFew[token.ChainID]; !ok {
			w.canWrapToFew[token.ChainID] = make(map[string]TokenInfo)
		}
		w.canWrapToFew[token.ChainID][token.UnwrapTokenAddress] = token

		if _, ok := w.isFewToken[token.ChainID]; !ok {
			w.isFewToken[token.ChainID] = make(map[string]TokenInfo)
		}
		w.isFewToken[token.ChainID][token.FewTokenAddress] = token
	}

	return w
}

func (w TokenWrapper) CanWrap(chainID valueobject.ChainID, address string) (shared.IWrapMetadata, bool) {
	value, ok := w.canWrapToFew[chainID][address]
	return value, ok
}

func (w TokenWrapper) IsWrapped(chainID valueobject.ChainID, address string) (shared.IWrapMetadata, bool) {
	value, ok := w.isFewToken[chainID][address]
	return value, ok
}
