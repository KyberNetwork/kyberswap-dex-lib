package kuruobv2

import (
	"context"
	"math/big"

	"github.com/KyberNetwork/ethrpc"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
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
	}

	var book L2Book
	var takerFeePps *big.Int
	var marketState uint8
	var protocolPaused bool
	req := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: p.Address, Method: "getL2Book",
			Params: []any{big.NewInt(bookDepth)}}, []any{&book}).
		AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: p.Address, Method: "takerFeePps"}, []any{&takerFeePps}).
		AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: p.Address, Method: "marketState"}, []any{&marketState})
	if t.config.AccountCore != "" {
		req.AddCall(&ethrpc.Call{ABI: kuruV2ABI, Target: t.config.AccountCore, Method: "protocolPaused"},
			[]any{&protocolPaused})
	}
	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	var extra Extra
	reserves := [2]*uint256.Int{new(uint256.Int), new(uint256.Int)}
	if marketState == marketStateLive && !protocolPaused {
		extra.TakerFeePps = takerFeePps.Uint64()
		extra.Bids = toLevels(book.BidPrices, book.BidSizes)
		extra.Asks = toLevels(book.AskPrices, book.AskSizes)
		// Reserves: base on the asks, quote on the bids (floor quote conversion like the sell walk).
		var tmp, quote uint256.Int
		quoteScale, pricePrecision := big256.TenPow(staticExtra.QuoteDecimals), uint256.NewInt(staticExtra.PricePrecision)
		for _, level := range extra.Asks {
			reserves[0].Add(reserves[0], tmp.Mul(level.Size, staticExtra.BaseSizeMultiplier))
		}
		for _, level := range extra.Bids {
			big256.MulDivDown(&quote, tmp.SetUint64(level.Price), level.Size, staticExtra.SizePrecision)
			reserves[1].Add(reserves[1], big256.MulDivDown(&quote, &quote, quoteScale, pricePrecision))
		}
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Reserves = entity.PoolReserves{reserves[0].Dec(), reserves[1].Dec()}
	p.Extra = string(extraBytes)
	p.BlockNumber = resp.BlockNumber.Uint64()
	return p, nil
}

// toLevels keeps the non-empty prefix of one L2 side (zero price pads the fixed-depth arrays).
func toLevels(prices []uint32, sizes []*big.Int) []Level {
	levels := make([]Level, 0, len(prices))
	for i, price := range prices {
		if price == 0 || sizes[i].Sign() <= 0 {
			break
		}
		levels = append(levels, Level{Price: uint64(price), Size: uint256.MustFromBig(sizes[i])})
	}
	return levels
}
