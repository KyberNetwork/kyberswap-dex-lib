package prismprop

import (
	"context"
	"errors"
	"math"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

var ErrGetOrderBookReverted = errors.New("getOrderBookWithParams reverted")

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

// GetNewPoolState makes one getOrderBookWithParams(token0, token1) call priced
// for our executor, so the levels already include the executor's taker spread
// (exactly what swap() settles) and no fee calibration is needed. Bids are the
// token0->token1 ladder and asks the token1->token0 ladder.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	l := logger.WithFields(logger.Fields{"poolAddress": p.Address, "dexID": t.config.DexID})
	l.Info("Start getting new state")

	token0, token1 := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)

	var res getOrderBookResult
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    routerABI,
		Target: t.config.RouterAddress,
		Method: methodGetOrderBookWithParams,
		Params: []any{token0, token1, QuoteParams{MsgSender: executorAddress, GasPrice: new(big.Int)}},
	}, []any{&res}).TryBlockAndAggregate()
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to aggregate RPC requests")
		return entity.Pool{}, err
	}
	if !resp.Result[0] {
		l.WithFields(logger.Fields{"error": ErrGetOrderBookReverted}).Error("failed to get order book")
		return entity.Pool{}, ErrGetOrderBookReverted
	}

	book := res.Book
	extra := orderbook.Extra{LevelsFrom: [2][]orderbook.Level{
		toLevels(book.Bids, true, p.Tokens[0].Decimals, p.Tokens[1].Decimals),
		toLevels(book.Asks, false, p.Tokens[1].Decimals, p.Tokens[0].Decimals),
	}}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		l.WithFields(logger.Fields{"error": err}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	// A token's reserve is what the book can pay out of it: asks pay base
	// (token0), bids pay quote (token1).
	var reserve0, reserve1 big.Int
	for _, lvl := range book.Asks.Levels {
		reserve0.Add(&reserve0, lvl.BaseAmount)
	}
	for _, lvl := range book.Bids.Levels {
		reserve1.Add(&reserve1, lvl.QuoteAmount)
	}

	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{reserve0.String(), reserve1.String()}
	p.SwapFee = 0 // the levels already include the taker spread; clears the old calibrated fee
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()

	l.Info("Finish updating state of pool")
	return p, nil
}

// toLevels converts one book side into order-book levels of the trade that
// pays tokenIn. inIsBase: bids take base in, asks take quote in. The book
// lists levels best price first, which is the order the contract fills them.
//
// order-book.NewPoolSimulatorWith takes minTrade from Levels[0].Size(), so the
// first level is a zero-size sentinel (same pattern as kuru-ob); the real
// minimum trade is 1 wei or a vault cap already applied to the levels.
// Returns nil for an empty side.
func toLevels(side BookSide, inIsBase bool, decimalsIn, decimalsOut uint8) []orderbook.Level {
	scaleIn, scaleOut := math.Pow10(int(decimalsIn)), math.Pow10(int(decimalsOut))
	levels := make([]orderbook.Level, 1, 1+len(side.Levels)) // levels[0] == zero-size sentinel
	for _, lvl := range side.Levels {
		in, out := lvl.BaseAmount, lvl.QuoteAmount
		if !inIsBase {
			in, out = out, in
		}
		inF, _ := in.Float64()
		outF, _ := out.Float64()
		if inF <= 0 {
			continue
		}
		size := inF / scaleIn
		levels = append(levels, orderbook.Level{size, outF / scaleOut / size})
	}
	if len(levels) == 1 {
		return nil
	}
	return levels
}
