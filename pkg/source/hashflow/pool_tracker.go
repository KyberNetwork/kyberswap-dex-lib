package hashflow

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/logger"
)

type PoolTracker struct {
	config *Config
	client IClient
}

func NewPoolTracker(cfg *Config, client IClient) *PoolTracker {
	return &PoolTracker{
		config: cfg,
		client: client,
	}
}

func (d *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	poolID, ok := ParsePoolID(p.Address)
	if !ok {
		var err = errors.New("failed to parse poolID")
		logger.
			WithFields(logger.Fields{"error": err, "poolID": poolID.String()}).
			Error("get new pool state failed > failed to parse poolID")
		return entity.Pool{}, err
	}

	pairs, err := d.client.ListPriceLevels(ctx, []string{poolID.MarketMaker})
	if err != nil {
		return entity.Pool{}, err
	}

	var findPair *Pair
	for _, pair := range pairs {
		pairPoolID := PoolID{
			MarketMaker: poolID.MarketMaker,
			Token0:      pair.Tokens[0],
			Token1:      pair.Tokens[1],
		}
		if pairPoolID == poolID {
			findPair = &pair
			break
		}
	}
	if findPair == nil {
		// Cannot find the current pair anymore, so disable this pair
		// by setting its reserves to zeroes.
		p.Extra = ""
		p.Reserves = entity.PoolReserves{"0", "0"}
		p.Timestamp = time.Now().Unix()
		return p, nil
	}

	extra := Extra{
		ZeroToOnePriceLevels: findPair.ZeroToOnePriceLevels,
		OneToZeroPriceLevels: findPair.OneToZeroPriceLevels,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.
			WithFields(logger.Fields{"error": err, "poolID": poolID.String()}).
			Error("get new pool state failed > marshal extra failed")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = calcReserves(*findPair)
	p.Timestamp = time.Now().Unix()

	return p, nil
}
