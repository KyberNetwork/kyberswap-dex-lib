package elfomofi

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	orderbook "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/order-book"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config        *Config
	ethrpcClient  *ethrpc.Client
	helperMu      sync.Mutex
	helperAddress string
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, pool.GetNewPoolStateParams{Logs: params.Logs}, params.Overrides)
}

func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
	_ map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	if len(p.Tokens) != 2 || p.Tokens[0] == nil || p.Tokens[1] == nil {
		return entity.Pool{}, fmt.Errorf("elfomofi: expected two pool tokens")
	}
	if t.config.Buffer <= 0 || t.config.Buffer > 10_000 {
		return entity.Pool{}, fmt.Errorf("elfomofi: buffer must be in (0, 10000] bps")
	}
	helperAddress, err := t.resolveHelperAddress(ctx)
	if err != nil {
		return entity.Pool{}, err
	}

	base, quote := common.HexToAddress(p.Tokens[0].Address), common.HexToAddress(p.Tokens[1].Address)
	var result getOrderbookResult
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{
		ABI:    helperABI,
		Target: helperAddress,
		Method: "getOrderbook",
		Params: []any{base, quote},
	}, []any{&result}).TryBlockAndAggregate()
	if err != nil {
		return entity.Pool{}, err
	}
	book, helperBlock, helperTimestamp := result.Book, result.BlockNumber, result.BlockTimestamp
	if book.Base != base || book.Quote != quote {
		return entity.Pool{}, fmt.Errorf("elfomofi: helper returned a different pair: %s/%s", book.Base, book.Quote)
	}
	if book.BalanceBase == nil || book.BalanceQuote == nil || book.BalanceBase.Sign() < 0 || book.BalanceQuote.Sign() < 0 {
		return entity.Pool{}, fmt.Errorf("elfomofi: helper returned invalid balances")
	}
	if helperBlock == nil || !helperBlock.IsUint64() || helperBlock.Sign() == 0 ||
		helperTimestamp == nil || !helperTimestamp.IsInt64() || helperTimestamp.Sign() <= 0 || resp.BlockNumber == nil ||
		helperBlock.Cmp(resp.BlockNumber) != 0 {
		return entity.Pool{}, fmt.Errorf("elfomofi: helper returned invalid block metadata")
	}

	// The helper probes at the pool's actual depth breaks. Its asks spend quote
	// for base, while bids spend base for quote; our pool tokens are base, quote.
	bids, err := cumulativeLevelsToOrderbook(book.BidCumulativeLevels, p.Tokens[0].Decimals, p.Tokens[1].Decimals, t.config.Buffer)
	if err != nil {
		return entity.Pool{}, err
	}
	asks, err := cumulativeLevelsToOrderbook(book.AskCumulativeLevels, p.Tokens[1].Decimals, p.Tokens[0].Decimals, t.config.Buffer)
	if err != nil {
		return entity.Pool{}, err
	}
	extraBytes, err := json.Marshal(orderbook.Extra{LevelsFrom: [2][]orderbook.Level{bids, asks}})
	if err != nil {
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = []string{book.BalanceBase.String(), book.BalanceQuote.String()}
	p.BlockNumber = helperBlock.Uint64()
	p.Timestamp = helperTimestamp.Int64()

	return p, nil
}

func (t *PoolTracker) resolveHelperAddress(ctx context.Context) (string, error) {
	if t.config.HelperAddress != "" {
		if !common.IsHexAddress(t.config.HelperAddress) || common.HexToAddress(t.config.HelperAddress) == (common.Address{}) {
			return "", fmt.Errorf("elfomofi: invalid helperAddress %q", t.config.HelperAddress)
		}
		return t.config.HelperAddress, nil
	}

	t.helperMu.Lock()
	defer t.helperMu.Unlock()
	if t.helperAddress != "" {
		return t.helperAddress, nil
	}
	chainID, err := t.ethrpcClient.GetETHClient().ChainID(ctx)
	if err != nil {
		return "", fmt.Errorf("elfomofi: get chain ID for helper: %w", err)
	}
	if !chainID.IsUint64() {
		return "", fmt.Errorf("elfomofi: unsupported chain ID %s", chainID)
	}
	helperAddress, ok := helperAddresses[chainID.Uint64()]
	if !ok {
		return "", fmt.Errorf("elfomofi: no helper address for chain ID %s", chainID)
	}
	t.helperAddress = helperAddress
	return helperAddress, nil
}

// cumulativeLevelsToOrderbook turns buffered cumulative outputs into marginal
// levels. Subtract in integer units before converting to floats so large
// cumulative values cannot erase small level deltas through cancellation.
func cumulativeLevelsToOrderbook(probes []CumulativeLevel, decimalsIn, decimalsOut uint8, buffer int64) ([]orderbook.Level, error) {
	if len(probes) == 0 {
		return nil, nil
	}
	levels := make([]orderbook.Level, 1, len(probes)+1) // zero-size sentinel
	var previousIn, previousOut big.Int
	bufferBI := big.NewInt(buffer)
	for _, probe := range probes {
		if probe.AmountIn == nil || probe.AmountOut == nil {
			return nil, fmt.Errorf("elfomofi: helper returned a nil cumulative level")
		}
		if probe.AmountIn.Sign() <= 0 || probe.AmountOut.Sign() <= 0 {
			continue
		}
		bufferedOut := new(big.Int).Mul(probe.AmountOut, bufferBI)
		bufferedOut.Div(bufferedOut, bignumber.BasisPoint)
		if probe.AmountIn.Cmp(&previousIn) <= 0 || bufferedOut.Cmp(&previousOut) <= 0 {
			continue
		}
		deltaIn := new(big.Int).Sub(probe.AmountIn, &previousIn)
		deltaOut := new(big.Int).Sub(bufferedOut, &previousOut)
		inFloat, _ := deltaIn.Float64()
		outFloat, _ := deltaOut.Float64()
		size := inFloat / math.Pow10(int(decimalsIn))
		price := outFloat / math.Pow10(int(decimalsOut)) / size
		if size <= 0 || price <= 0 || math.IsInf(size, 0) || math.IsInf(price, 0) || math.IsNaN(price) {
			return nil, fmt.Errorf("elfomofi: helper returned an unrepresentable cumulative level")
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
