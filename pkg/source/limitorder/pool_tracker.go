package limitorder

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/KyberNetwork/logger"
	"golang.org/x/sync/errgroup"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
)

type PoolTracker struct {
	config           *Config
	limitOrderClient *httpClient
}

func NewPoolTracker(cfg *Config) *PoolTracker {
	limitOrderClient := NewHTTPClient(cfg.LimitOrderHTTPUrl)

	return &PoolTracker{
		config:           cfg,
		limitOrderClient: limitOrderClient,
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[LimitOrder] Start getting new states for pool %v", p.Address)
	if len(p.Tokens) < 2 {
		err := errors.New("number of token should be greater than or equal 2")
		logger.Errorf(err.Error())
		return entity.Pool{}, err
	}
	token0, token1 := p.Tokens[0], p.Tokens[1]
	if strings.ToLower(token0.Address) < strings.ToLower(token1.Address) {
		token0, token1 = p.Tokens[1], p.Tokens[0]
	}

	var contractAddress string
	if d.config.SupportMultiSCs {
		var staticExtra StaticExtra
		if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
			return entity.Pool{}, err
		}
		contractAddress = staticExtra.ContractAddress
	} else {
		contractAddress = ""
	}

	extra := Extra{}
	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		buyOrders, err := d.limitOrderClient.ListOrders(gCtx, listOrdersFilter{
			ChainID:         ChainID(d.config.ChainID),
			MakerAsset:      token0.Address,
			TakerAsset:      token1.Address,
			ContractAddress: contractAddress,
		})
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to get listOrders for buy side")
			return err
		}
		extra.BuyOrders = buyOrders
		return nil
	})

	g.Go(func() error {
		sellOrders, err := d.limitOrderClient.ListOrders(gCtx, listOrdersFilter{
			ChainID:         ChainID(d.config.ChainID),
			MakerAsset:      token1.Address,
			TakerAsset:      token0.Address,
			ContractAddress: contractAddress,
		})
		if err != nil {
			logger.WithFields(logger.Fields{
				"error": err,
			}).Errorf("failed to get listOrders for sell side")
			return err
		}
		extra.SellOrders = sellOrders
		return nil
	})
	err := g.Wait()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to get extra data")
		return entity.Pool{}, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	return p, nil
}
