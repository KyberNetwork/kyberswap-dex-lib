package kuruobv2

import (
	"context"
	"math/big"
	"sync/atomic"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/valueobject"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	limiter      atomic.Pointer[common.Address] // AccountCore's WithdrawalLimiter; immutable once bound
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}
}

func (t *PoolTracker) GetNewPoolState(ctx context.Context, p entity.Pool,
	_ poolpkg.GetNewPoolStateParams) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	} else if staticExtra.SizePrecision == nil || staticExtra.BaseSizeMultiplier == nil {
		return p, ErrInvalidStaticExtra
	}

	limiter, err := t.withdrawalLimiter(ctx)
	if err != nil {
		return p, err
	}

	var book L2Book
	var takerFeeCap, defaultTakerFee *big.Int
	var marketState uint8
	var protocolPaused bool
	req := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: p.Address, Method: "getL2Book",
			Params: []any{big.NewInt(bookDepth)}}, []any{&book}).
		AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: p.Address, Method: "takerFeePps"}, []any{&takerFeeCap}).
		AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: p.Address, Method: "marketState"}, []any{&marketState}).
		AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: t.config.AccountCore, Method: "defaultSpotTakerFeePps"},
			[]any{&defaultTakerFee}).
		AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: t.config.AccountCore, Method: "protocolPaused"},
			[]any{&protocolPaused})
	// Every swap withdraws both tokens (output and unused input). The limiter's price check reverts
	// InvalidWithdrawalPrice when a token's feed is stale (e.g. off-hours), so such a market can't settle.
	for _, token := range p.Tokens {
		addr := common.HexToAddress(token.Address)
		if staticExtra.HasNative && valueobject.IsWrappedNative(token.Address, t.config.ChainId) {
			addr = common.Address{} // AccountCore keys native as address(0)
		}
		req.AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: limiter.Hex(), Method: "requireFreshPrice",
			Params: []any{addr}}, []any{&struct{}{}}) // void: ethrpc still indexes Output
	}
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	for _, ok := range resp.Result[:numStateCalls] {
		if !ok {
			return p, ErrCallFailed
		}
	}
	feedsFresh := resp.Result[numStateCalls] && resp.Result[numStateCalls+1]

	var extra Extra
	var baseReserve, quoteReserve uint256.Int
	if marketState == marketStateLive && !protocolPaused && feedsFresh {
		// The market value is only a cap; AccountCore charges min(default tier, cap) (matches
		// effectiveSpotTakerFeePps for accounts without a tier).
		extra.TakerFeePps = min(takerFeeCap.Uint64(), defaultTakerFee.Uint64())
		extra.Bids = toLevels(book.BidPrices, book.BidSizes)
		extra.Asks = toLevels(book.AskPrices, book.AskSizes)
		// Reserves: base on the asks, quote on the bids (floor quote conversion like the sell walk).
		var tmp, quote uint256.Int
		quoteScale, pricePrecision := big256.TenPow(staticExtra.QuoteDecimals), tmp.SetUint64(staticExtra.PricePrecision)
		for i := range extra.Asks {
			baseReserve.Add(&baseReserve, quote.Mul(&extra.Asks[i].Size, staticExtra.BaseSizeMultiplier))
		}
		for i := range extra.Bids {
			// price < 2^32 and size < 2^128: the product fits.
			quote.Mul(quote.SetUint64(extra.Bids[i].Price), &extra.Bids[i].Size).Div(&quote, staticExtra.SizePrecision)
			quoteReserve.Add(&quoteReserve, big256.MulDivDown(&quote, &quote, quoteScale, pricePrecision))
		}
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{baseReserve.Dec(), quoteReserve.Dec()}
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()
	return p, nil
}

// withdrawalLimiter reads (once) the limiter AccountCore routes withdrawals through.
func (t *PoolTracker) withdrawalLimiter(ctx context.Context) (common.Address, error) {
	if l := t.limiter.Load(); l != nil {
		return *l, nil
	}
	var l common.Address
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).AddCall(&ethrpc.Call{ABI: kuruV2ABI,
		Target: t.config.AccountCore, Method: "withdrawalLimiter"}, []any{&l}).Call(); err != nil {
		return l, err
	}
	t.limiter.Store(&l)
	return l, nil
}

// toLevels keeps the non-empty prefix of one L2 side (zero price pads the fixed-depth arrays).
func toLevels(prices []uint32, sizes []*big.Int) []Level {
	levels := make([]Level, 0, len(prices))
	for i, price := range prices {
		if price == 0 || sizes[i].Sign() <= 0 {
			break
		}
		var size uint256.Int
		if size.SetFromBig(sizes[i]) || size.BitLen() > 128 { // book sizes are uint128
			break
		}
		levels = append(levels, Level{Price: uint64(price), Size: size})
	}
	return levels
}
