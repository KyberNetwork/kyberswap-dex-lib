package ezeth

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	extra, blockNumber, err := getExtra(ctx, t.ethrpcClient)
	if err != nil {
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	tokens := []*entity.PoolToken{
		{
			Address:   strings.ToLower(EzEthToken),
			Symbol:    "ezETH",
			Decimals:  18,
			Name:      "Renzo Restaked ETH",
			Swappable: true,
		},
		{
			Address:   strings.ToLower(WETH),
			Symbol:    "WETH",
			Decimals:  18,
			Name:      "Wrapped Ether",
			Swappable: true,
		},
	}
	tokens = append(tokens, extra.collaterals...)
	reserves := make([]string, len(extra.collaterals)+1)
	for i := 0; i < len(reserves); i++ {
		reserves[i] = defaultReserves
	}

	p.Tokens = tokens
	p.Reserves = reserves
	p.Extra = string(extraBytes)
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}
