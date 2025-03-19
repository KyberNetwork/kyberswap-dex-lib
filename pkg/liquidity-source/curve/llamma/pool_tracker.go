package llamma

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/curve/shared"
	big256 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/big256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	poolpkg "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	logger       logger.Logger
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) *PoolTracker {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		logger: logger.WithFields(logger.Fields{
			"dexId":   config.DexID,
			"dexType": DexType,
		}),
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ poolpkg.GetNewPoolStateParams,
) (entity.Pool, error) {
	lg := t.logger.WithFields(logger.Fields{"poolAddress": p.Address})
	lg.Info("Start updating state...")

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var (
		basePrice   *big.Int
		priceOracle *big.Int
		fee         *big.Int
		adminFee    *big.Int
		adminFeesX  *big.Int
		adminFeesY  *big.Int
		activeBand  *big.Int
		minBand     *big.Int
		maxBand     *big.Int

		balances = make([]*big.Int, 2)
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodGetBasePrice,
	}, []any{&basePrice})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodPriceOracle,
	}, []any{&priceOracle})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodFee,
	}, []any{&fee})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodAdminFee,
	}, []any{&adminFee})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodAdminFeesX,
	}, []any{&adminFeesX})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodAdminFeesY,
	}, []any{&adminFeesY})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodActiveBand,
	}, []any{&activeBand})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodMinBand,
	}, []any{&minBand})
	calls.AddCall(&ethrpc.Call{
		ABI:    curveLlammaABI,
		Target: p.Address,
		Method: llammaMethodMaxBand,
	}, []any{&maxBand})
	calls.AddCall(&ethrpc.Call{
		ABI:    shared.ERC20ABI,
		Target: p.Tokens[0].Address,
		Method: shared.ERC20MethodBalanceOf,
		Params: []interface{}{common.HexToAddress(p.Address)},
	}, []any{&balances[0]})
	calls.AddCall(&ethrpc.Call{
		ABI:    shared.ERC20ABI,
		Target: p.Tokens[1].Address,
		Method: shared.ERC20MethodBalanceOf,
		Params: []interface{}{common.HexToAddress(p.Address)},
	}, []any{&balances[1]})
	resp, err := calls.Aggregate()
	if err != nil {
		return p, err
	}

	bands, err := t.getBands(ctx, p.Address, activeBand.Int64(), minBand.Int64(), maxBand.Int64(), t.config.MaxBandLimit)
	if err != nil {
		return p, err
	}
	availableBalances := t.calcAvailableBalances(p.Tokens, bands)

	extraBytes, err := json.Marshal(&Extra{
		BasePrice:         uint256.MustFromBig(basePrice),
		PriceOracle:       uint256.MustFromBig(priceOracle),
		Fee:               uint256.MustFromBig(fee),
		AdminFee:          uint256.MustFromBig(adminFee),
		AdminFeesX:        uint256.MustFromBig(adminFeesX),
		AdminFeesY:        uint256.MustFromBig(adminFeesY),
		ActiveBand:        activeBand.Int64(),
		MinBand:           minBand.Int64(),
		MaxBand:           maxBand.Int64(),
		Bands:             bands,
		AvailableBalances: availableBalances,
	})
	if err != nil {
		lg.WithFields(logger.Fields{
			"error": err,
		}).Error("failed to marshal extra data")
		return entity.Pool{}, err
	}

	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.Reserves = entity.PoolReserves{
		new(big.Int).Sub(balances[0], adminFeesX).String(),
		new(big.Int).Sub(balances[1], adminFeesY).String(),
	}
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}

	lg.WithFields(logger.Fields{
		"balances":          p.Reserves,
		"availableBalances": availableBalances,
	}).Info("Finish updating state.")

	return p, nil
}

func (t *PoolTracker) getBands(
	ctx context.Context,
	poolAddress string, activeBand, minBand, maxBand, bandLimit int64,
) ([]Band, error) {
	startBand := activeBand - (bandLimit+1)/2
	if startBand < minBand {
		startBand = minBand
	}

	endBand := startBand + bandLimit - 1
	if endBand > maxBand {
		endBand = maxBand
	}

	bandCount := endBand - startBand + 1
	var (
		bandsX = make([]*big.Int, bandCount)
		bandsY = make([]*big.Int, bandCount)
	)

	calls := t.ethrpcClient.NewRequest().SetContext(ctx)
	for i := int64(0); i < bandCount; i++ {
		bandIndex := big.NewInt(i + startBand)
		calls.AddCall(&ethrpc.Call{
			ABI:    curveLlammaABI,
			Target: poolAddress,
			Method: llammaMethodBandsX,
			Params: []interface{}{bandIndex},
		}, []any{&bandsX[i]})
		calls.AddCall(&ethrpc.Call{
			ABI:    curveLlammaABI,
			Target: poolAddress,
			Method: llammaMethodBandsY,
			Params: []interface{}{bandIndex},
		}, []any{&bandsY[i]})
	}
	if _, err := calls.Aggregate(); err != nil {
		return nil, err
	}

	bands := make([]Band, 0, bandCount)
	for i := int64(0); i < bandCount; i++ {
		if bandsX[i].Sign() == 0 && bandsY[i].Sign() == 0 {
			continue
		}

		bands = append(bands, Band{
			Index: i + startBand,
			BandX: uint256.MustFromBig(bandsX[i]),
			BandY: uint256.MustFromBig(bandsY[i]),
		})
	}

	return bands, nil
}

func (t *PoolTracker) calcAvailableBalances(tokens []*entity.PoolToken, bands []Band) []*uint256.Int {
	totalX := uint256.NewInt(0)
	totalY := uint256.NewInt(0)
	for _, band := range bands {
		totalX.Add(totalX, band.BandX)
		totalY.Add(totalY, band.BandY)
	}

	precisionX := big256.TenPowInt(18 - tokens[0].Decimals)
	precisionY := big256.TenPowInt(18 - tokens[1].Decimals)

	totalX.Div(totalX, precisionX)
	totalY.Div(totalY, precisionY)

	return []*uint256.Int{totalX, totalY}
}
