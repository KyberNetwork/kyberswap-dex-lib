package token

import (
	"context"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	routerEntity "github.com/KyberNetwork/router-service/internal/pkg/entity"
	"github.com/KyberNetwork/router-service/internal/pkg/valueobject"
)

type IFallbackRepository interface {
	FindByAddresses(ctx context.Context, addresses []string) ([]*entity.Token, error)
	FindTokenInfoByAddress(ctx context.Context, chainID valueobject.ChainID, addresses []string) ([]*routerEntity.TokenInfo, error)
}

type ITokenAPI interface {
	FindTokenInfos(ctx context.Context, chainID valueobject.ChainID, addresses []string) ([]*routerEntity.TokenInfo, error)
}
