package reclamm

import (
	"github.com/KyberNetwork/ethrpc"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/balancer/v3/shared"
	poollist "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/list"
	graphqlpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/graphql"
)

var _ = poollist.RegisterFactoryCEG(DexType, NewPoolsListUpdater)

func NewPoolsListUpdater(config *shared.Config, ethrpcClient *ethrpc.Client,
	graphqlClient *graphqlpkg.Client) *shared.PoolsListUpdater {
	config.PoolType = DexType
	config.SubgraphPoolType = SubgraphPoolType
	return shared.NewPoolsListUpdater(config, ethrpcClient, graphqlClient)
}
