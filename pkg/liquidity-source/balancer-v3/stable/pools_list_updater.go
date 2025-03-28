package stable

import (
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer-v3/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

var _ = poollist.RegisterFactoryCG(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *shared.Config, graphqlClient *graphqlpkg.Client) *shared.PoolsListUpdater {
	config.PoolType = DexType
	config.SubgraphPoolType = SubgraphPoolType
	return shared.NewPoolsListUpdater(config, graphqlClient)
}
