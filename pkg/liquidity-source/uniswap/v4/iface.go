package uniswapv4

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4/shared"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type ITokenWrapper interface {
	// CanWrap checks if the token can be wrapped on the given chain.
	// Return the wrap metadata, and a boolean indicating if wrapping is possible.
	CanWrap(chain valueobject.ChainID, token string) (shared.IWrapMetadata, bool)

	// IsWrapped checks if the token is already wrapped on the given chain.
	// Return the unwrap metadata, and a boolean indicating if the token is wrapped.
	IsWrapped(chain valueobject.ChainID, token string) (shared.IWrapMetadata, bool)
}
