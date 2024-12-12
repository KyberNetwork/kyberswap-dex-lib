package virtualfun

import (
	"context"
	"encoding/json"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, params, nil)
}

func (d *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return d.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (d *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	if !p.Tokens[0].Swappable && !p.Tokens[1].Swappable {
		return p, nil
	}

	logger.WithFields(logger.Fields{"pool_id": p.Address}).Info("Started getting new pool state")

	tokenReserves, pairReserves, canPoolTradable, blockNumber, err := d.getReserves(ctx, p.Address, p.Tokens, overrides)
	if err != nil {
		return p, err
	}

	if p.BlockNumber > blockNumber.Uint64() {
		return p, nil
	}

	// Disable pool : need a solution to clear these pools
	if !canPoolTradable {
		p.Tokens[0].Swappable = false
		p.Tokens[1].Swappable = false
		p.Reserves[0] = "0"
		p.Reserves[1] = "0"

		return p, nil
	}

	newReserves := make(entity.PoolReserves, 0, len(tokenReserves))
	for _, reserve := range tokenReserves {
		if reserve == nil {
			newReserves = append(newReserves, "0")
		} else {
			newReserves = append(newReserves, reserve.String())
		}
	}

	buyTax, sellTax, kLast, err := d.getTax(ctx, p.Address, blockNumber)
	if err != nil {
		return p, err
	}

	var extra = Extra{
		SellTax:  sellTax,
		BuyTax:   buyTax,
		ReserveA: pairReserves[0],
		ReserveB: pairReserves[1],
		KLast:    kLast,
	}

	newExtra, err := json.Marshal(&extra)
	if err != nil {
		return p, err
	}

	p.Reserves = newReserves
	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	p.Extra = string(newExtra)

	return p, nil
}

func (d *PoolTracker) getReserves(
	ctx context.Context,
	poolAddress string,
	tokens []*entity.PoolToken,
	overrides map[common.Address]gethclient.OverrideAccount,
) ([]*big.Int, [2]*big.Int, bool, *big.Int, error) {
	var (
		tokenReserves = make([]*big.Int, len(tokens))
		pairReserves  [2]*big.Int
		tradable      = true
	)

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	if overrides != nil {
		req.SetOverrides(overrides)
	}

	// Fetch individual token balances for the pool
	for i, token := range tokens {
		req.AddCall(&ethrpc.Call{
			ABI:    erc20ABI,
			Target: token.Address,
			Method: erc20BalanceOfMethod,
			Params: []interface{}{common.HexToAddress(poolAddress)},
		}, []interface{}{&tokenReserves[i]})
	}

	// Fetch pair reserves used for AMM calculations
	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: pairGetReservesMethod,
		Params: nil,
	}, []interface{}{&pairReserves})

	// Call to detect if pool can tradable ? Tradable if there is an error
	req.AddCall(&ethrpc.Call{
		ABI:    bondingABI,
		Target: d.config.BondingAddress,
		Method: bondingUnwrapTokenMethod,
		Params: []interface{}{common.HexToAddress(tokens[0].Address), []common.Address{}},
	}, []interface{}{&struct{}{}})

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, [2]*big.Int{}, tradable, nil, err
	}

	// Check the last call result
	if resp.Result[len(resp.Result)-1] {
		tradable = false
	}

	return tokenReserves, pairReserves, tradable, resp.BlockNumber, nil
}

func (d *PoolTracker) getTax(ctx context.Context, poolAddress string, blocknumber *big.Int) (*big.Int, *big.Int, *big.Int, error) {
	var (
		buyTax, sellTax = bignumber.ZeroBI, bignumber.ZeroBI
		kLast           = bignumber.ZeroBI
	)

	req := d.ethrpcClient.NewRequest().SetContext(ctx)

	if blocknumber != nil {
		req.SetBlockNumber(blocknumber)
	}

	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: factoryBuyTaxMethod,
		Params: nil,
	}, []interface{}{&buyTax})
	req.AddCall(&ethrpc.Call{
		ABI:    factoryABI,
		Target: d.config.FactoryAddress,
		Method: factorySellTaxMethod,
		Params: nil,
	}, []interface{}{&sellTax})
	req.AddCall(&ethrpc.Call{
		ABI:    pairABI,
		Target: poolAddress,
		Method: pairKLastMethod,
		Params: nil,
	}, []interface{}{&kLast})

	if _, err := req.Aggregate(); err != nil {
		return nil, nil, nil, err
	}

	return buyTax, sellTax, kLast, nil
}
