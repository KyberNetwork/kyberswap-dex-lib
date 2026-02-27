package wcm

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltracker "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/goccy/go-json"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
	blockNumber  *big.Int
}

var _ = pooltracker.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	lg := logger.WithFields(logger.Fields{
		"dexId":   config.DexID,
		"dexType": DexType,
	})

	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logger:       lg,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	t.logger.Infof("Start getting new pool state for pool: %s", p.Address)

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		t.logger.Errorf("failed to unmarshal static extra: %v", err)
		return p, err
	}

	orderBook, err := t.getOrderBook(ctx, p.Address)
	if err != nil {
		t.logger.Errorf("failed to get order book: %v", err)
		return p, err
	}

	extra := Extra{
		OrderBook: orderBook,
	}

	extra.TakerFeeMultiplier, extra.FromMaxFee, extra.ToMaxFee, extra.IsHalted, extra.MinOrderQuantity, err = t.fetchDynamicParams(ctx, p.Address)
	if err != nil {
		t.logger.Errorf("failed to fetch dynamic params: %v", err)
		return p, err
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		t.logger.Errorf("failed to marshal extra data: %v", err)
		return p, err
	}

	baseReserve, quoteReserve := t.calculateLiquidity(orderBook, staticExtra, p.Tokens)

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = []string{
		baseReserve.String(),  // Base token available to buy (from asks)
		quoteReserve.String(), // Quote token available to buy (from bids)
	}

	if t.blockNumber != nil {
		p.BlockNumber = t.blockNumber.Uint64()
	}

	t.logger.Infof("Successfully updated pool state for pool: %s", p.Address)

	return p, nil
}

func (t *PoolTracker) getOrderBook(ctx context.Context, orderBookAddr string) (OrderBook, error) {
	maxCount := uint32(maxOrderBookLevels)

	var bestBidOffer *big.Int

	orderBook := OrderBook{
		Bids: make([]OrderBookLevel, 0, maxCount),
		Asks: make([]OrderBookLevel, 0, maxCount),
	}

	decodeLevels := func(raw []*big.Int, out *[]OrderBookLevel) uint64 {
		if len(raw) == 0 || raw[0] == nil {
			return 0
		}
		header := raw[0]
		headerVal := header.Uint64()
		count := int(headerVal & 0xFFFFFFFF)
		nextRestart := headerVal >> 32
		if count > len(raw)-1 {
			count = len(raw) - 1
		}
		bucketCount := (count + 1) / 2
		if bucketCount > len(raw)-1 {
			bucketCount = len(raw) - 1
		}
		decoded := 0
		for i := 1; i <= bucketCount && decoded < count; i++ {
			p1, q1, p2, q2 := decodeDepthChartBucket(raw[i])
			if p1.Sign() > 0 || q1.Sign() > 0 {
				*out = append(*out, OrderBookLevel{Price: p1, Quantity: q1})
			}
			decoded++
			if decoded >= count {
				break
			}
			if p2.Sign() > 0 || q2.Sign() > 0 {
				*out = append(*out, OrderBookLevel{Price: p2, Quantity: q2})
			}
			decoded++
		}
		return uint64(nextRestart)
	}

	buyRestart := uint64(0)
	sellRestart := uint64(0)
	buyDone := false
	sellDone := false
	needBestBidOffer := true

	for !buyDone || !sellDone || needBestBidOffer {
		var buyRaw, sellRaw []*big.Int
		req := t.ethrpcClient.NewRequest().SetContext(ctx)

		if needBestBidOffer {
			req.AddCall(&ethrpc.Call{
				ABI:    spotOrderBookABI,
				Target: orderBookAddr,
				Method: "bestBidOffer",
				Params: nil,
			}, []any{&bestBidOffer})
		}
		if !buyDone {
			req.AddCall(&ethrpc.Call{
				ABI:    spotOrderBookABI,
				Target: orderBookAddr,
				Method: "retrieveBuyDepthChart",
				Params: []any{maxCount, buyRestart},
			}, []any{&buyRaw})
		}
		if !sellDone {
			req.AddCall(&ethrpc.Call{
				ABI:    spotOrderBookABI,
				Target: orderBookAddr,
				Method: "retrieveSellDepthChart",
				Params: []any{maxCount, sellRestart},
			}, []any{&sellRaw})
		}

		resp, err := req.Aggregate()
		if err != nil {
			return OrderBook{}, err
		}

		t.blockNumber = resp.BlockNumber

		needBestBidOffer = false

		if !buyDone {
			next := decodeLevels(buyRaw, &orderBook.Bids)
			if next == 0 || next == buyRestart {
				buyDone = true
			} else {
				buyRestart = next
			}
		}
		if !sellDone {
			next := decodeLevels(sellRaw, &orderBook.Asks)
			if next == 0 || next == sellRestart {
				sellDone = true
			} else {
				sellRestart = next
			}
		}
	}

	// When depth charts gave no levels, use best bid/ask from bestBidOffer
	if (len(orderBook.Bids) == 0 || len(orderBook.Asks) == 0) && bestBidOffer != nil && bestBidOffer.Sign() > 0 {
		sellPrice, sellQty, buyPrice, buyQty := decodeBestBidOffer(bestBidOffer)
		if len(orderBook.Bids) == 0 && buyPrice != nil && buyPrice.Sign() > 0 && buyQty != nil && buyQty.Sign() > 0 {
			orderBook.Bids = append(orderBook.Bids, OrderBookLevel{Price: buyPrice, Quantity: buyQty})
		}
		if len(orderBook.Asks) == 0 && sellPrice != nil && sellPrice.Sign() > 0 && sellQty != nil && sellQty.Sign() > 0 {
			orderBook.Asks = append(orderBook.Asks, OrderBookLevel{Price: sellPrice, Quantity: sellQty})
		}
	}

	return orderBook, nil
}

func (t *PoolTracker) fetchDynamicParams(ctx context.Context, poolAddress string) (*big.Int, *big.Int, *big.Int, bool, *big.Int, error) {
	var orderBookConfigPacked *big.Int
	var isHalted bool
	var minOrderQuantity *big.Int

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    compositeExchangeABI,
		Target: t.config.ExchangeAddress,
		Method: "readOrderBookConfig",
		Params: []any{common.HexToAddress(poolAddress)},
	}, []any{&orderBookConfigPacked})

	req.AddCall(&ethrpc.Call{
		ABI:    spotOrderBookABI,
		Target: poolAddress,
		Method: "isHalted",
		Params: nil,
	}, []any{&isHalted})

	req.AddCall(&ethrpc.Call{
		ABI:    spotOrderBookABI,
		Target: poolAddress,
		Method: "readMinOrderQuantity",
		Params: nil,
	}, []any{&minOrderQuantity})

	resp, err := req.Aggregate()
	if err != nil {
		return nil, nil, nil, false, nil, err
	}

	t.blockNumber = resp.BlockNumber

	takerFeeRaw, fromMaxFee, toMaxFee := UnpackOrderBookConfig(orderBookConfigPacked)

	return takerFeeRaw, fromMaxFee, toMaxFee, isHalted, minOrderQuantity, nil
}

func (t *PoolTracker) calculateLiquidity(orderBook OrderBook, staticExtra StaticExtra, tokens []*entity.PoolToken) (*big.Int, *big.Int) {
	buyDec := uint8(18)
	payDec := uint8(18)
	if len(tokens) >= 2 {
		buyDec = tokens[0].Decimals
		payDec = tokens[1].Decimals
	}

	baseReserve := new(big.Int)
	quoteReserve := new(big.Int)

	// Reserves[0] (Base) is what taker can GET = sum of asks
	for _, ask := range orderBook.Asks {
		askQtyRaw := scaleAmountDecimals(ask.Quantity, staticExtra.BuyTokenPositionDecimals, buyDec)
		baseReserve.Add(baseReserve, askQtyRaw)
	}

	// Reserves[1] (Quote) is what taker can GET = sum of (bids * prices)
	for _, bid := range orderBook.Bids {
		quoteAmount := new(big.Int).Mul(bid.Quantity, bid.Price)
		quoteAmount.Div(quoteAmount, PricePrecisionMultiplier)
		quoteAmountRaw := scaleAmountDecimals(quoteAmount, staticExtra.BuyTokenPositionDecimals, payDec)
		quoteReserve.Add(quoteReserve, quoteAmountRaw)
	}

	return baseReserve, quoteReserve
}
