package syntropiaprop

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &PoolTracker{config: cfg, ethrpcClient: ethrpcClient}, nil
}

// GetNewPoolState reads the pair's depth in one call of the lens. Prices are pushed without
// an event, so there is nothing in the logs to follow.
func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if len(p.Tokens) != 2 || p.Tokens[0] == nil || p.Tokens[1] == nil {
		return entity.Pool{}, ErrInvalidPool
	}

	base, quote := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)
	var result getOrderbookResult
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: t.config.Lens,
		Method: "getOrderbook",
		Params: []any{base, quote},
	}, []any{&result}).TryBlockAndAggregate()
	if err != nil {
		return entity.Pool{}, err
	}

	book, blockNumber, blockTimestamp := result.Book, result.BlockNumber, result.BlockTimestamp
	if book.Base != base || book.Quote != quote {
		return entity.Pool{}, fmt.Errorf("%w: pair %s/%s", ErrInvalidBook, book.Base, book.Quote)
	}
	if book.BalanceBase == nil || book.BalanceQuote == nil {
		return entity.Pool{}, fmt.Errorf("%w: no balances", ErrInvalidBook)
	}
	if blockNumber == nil || resp.BlockNumber == nil || blockNumber.Cmp(resp.BlockNumber) != 0 ||
		blockTimestamp == nil || !blockTimestamp.IsInt64() || blockTimestamp.Sign() <= 0 {
		return entity.Pool{}, fmt.Errorf("%w: block metadata", ErrInvalidBook)
	}

	// Bids spend base (token 0) and asks spend quote (token 1).
	bids, err := levelsToOrderbook(book.BidCumulativeLevels, p.Tokens[0].Decimals, p.Tokens[1].Decimals, t.config.Buffer)
	if err != nil {
		return entity.Pool{}, err
	}
	asks, err := levelsToOrderbook(book.AskCumulativeLevels, p.Tokens[1].Decimals, p.Tokens[0].Decimals, t.config.Buffer)
	if err != nil {
		return entity.Pool{}, err
	}
	extraBytes, err := json.Marshal(orderbook.Extra{LevelsFrom: [2][]orderbook.Level{bids, asks}})
	if err != nil {
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{book.BalanceBase.String(), book.BalanceQuote.String()}
	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = blockTimestamp.Int64()

	return p, nil
}

// levelsToOrderbook turns the lens's cumulative probes into marginal levels, each at the
// average price of its stretch, after keeping `buffer` bps of every output. The pool's price
// only worsens with size, so a part of a stretch is quoted no better than the pool pays.
// The subtraction is done on integers so that a large total cannot swallow a small step.
func levelsToOrderbook(probes []Level, decimalsIn, decimalsOut uint8, buffer int64) ([]orderbook.Level, error) {
	levels := make([]orderbook.Level, 1, len(probes)+1) // a zero-size level: no minimum trade
	var previousIn, previousOut big.Int
	bufferBI := big.NewInt(buffer)
	for _, probe := range probes {
		if probe.AmountIn == nil || probe.AmountOut == nil {
			return nil, fmt.Errorf("%w: nil level", ErrInvalidBook)
		}
		bufferedOut := new(big.Int).Mul(probe.AmountOut, bufferBI)
		bufferedOut.Div(bufferedOut, bignumber.BasisPoint)
		if probe.AmountIn.Cmp(&previousIn) <= 0 || bufferedOut.Cmp(&previousOut) <= 0 {
			continue
		}
		deltaIn, _ := new(big.Int).Sub(probe.AmountIn, &previousIn).Float64()
		deltaOut, _ := new(big.Int).Sub(bufferedOut, &previousOut).Float64()
		size := deltaIn / math.Pow10(int(decimalsIn))
		price := deltaOut / math.Pow10(int(decimalsOut)) / size
		if size <= 0 || price <= 0 || math.IsInf(size, 0) || math.IsInf(price, 0) || math.IsNaN(price) {
			return nil, fmt.Errorf("%w: level cannot be represented", ErrInvalidBook)
		}
		levels = append(levels, orderbook.Level{size, price})
		previousIn.Set(probe.AmountIn)
		previousOut.Set(bufferedOut)
	}
	if len(levels) == 1 {
		return nil, nil
	}
	return levels, nil
}
