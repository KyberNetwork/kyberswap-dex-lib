package generatepath

import (
	"math/big"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

type genBestPathsTask struct {
	tokenIn   string
	tokenOuts []string
	amountIn  *big.Int
}

type genBestPathsResult struct {
	tokenIn             string
	bestPathsByTokenOut map[string][]*entity.MinimalPath
}
