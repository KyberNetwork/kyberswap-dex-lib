package ponsv2

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(config *Config, ethrpcClient *ethrpc.Client) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{"address": p.Address}).Info("start getting new state of pons-v2 curve")

	var (
		reserves       curveReservesResult
		graduated      bool
		feeBps         *big.Int
		creatorTaxBps  *big.Int
		reservedTokens *big.Int
		isNativeQuote  bool
	)

	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    curveABI,
		Target: p.Address,
		Method: curveMethodGetReserves,
	}, []any{&reserves})
	req.AddCall(&ethrpc.Call{
		ABI:    curveABI,
		Target: p.Address,
		Method: curveMethodGraduated,
	}, []any{&graduated})
	// FeeBps/CreatorTaxBps/ReservedTokens/IsNativeQuote are immutable
	// on-chain, but re-fetched every pass rather than only once at discovery:
	// discovery (pool_factory.go's DecodePoolCreated) is a pure log decode
	// with no RPC calls, so this is the first point any of them are known.
	req.AddCall(&ethrpc.Call{
		ABI:    curveABI,
		Target: p.Address,
		Method: curveMethodFeeBps,
	}, []any{&feeBps})
	req.AddCall(&ethrpc.Call{
		ABI:    curveABI,
		Target: p.Address,
		Method: curveMethodCreatorTax,
	}, []any{&creatorTaxBps})
	req.AddCall(&ethrpc.Call{
		ABI:    curveABI,
		Target: p.Address,
		Method: curveMethodReserved,
	}, []any{&reservedTokens})
	req.AddCall(&ethrpc.Call{
		ABI:    curveABI,
		Target: p.Address,
		Method: curveMethodIsNative,
	}, []any{&isNativeQuote})

	resp, err := req.Aggregate()
	if err != nil {
		return p, err
	}

	if reserves.QuoteReserve == nil || reserves.TokenReserve == nil ||
		feeBps == nil || creatorTaxBps == nil || reservedTokens == nil {
		return p, ErrInvalidReserve
	}

	quoteReserve := uint256.MustFromBig(reserves.QuoteReserve)
	tokenReserve := uint256.MustFromBig(reserves.TokenReserve)

	extra := Extra{
		QuoteReserve: quoteReserve,
		TokenReserve: tokenReserve,
		Graduated:    graduated,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	staticExtra := StaticExtra{
		FeeBps:         uint16(feeBps.Uint64()),
		CreatorTaxBps:  uint16(creatorTaxBps.Uint64()),
		ReservedTokens: uint256.MustFromBig(reservedTokens),
		IsNativeQuote:  isNativeQuote,
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.StaticExtra = string(staticExtraBytes)
	p.Timestamp = time.Now().Unix()
	if resp.BlockNumber != nil {
		p.BlockNumber = resp.BlockNumber.Uint64()
	}
	// Reserves report the FULL reserve pair from getReserves() (matching what
	// the curve's own constant-product math is priced against); the sellable
	// ceiling below that (TokenReserve - StaticExtra.ReservedTokens) is
	// derived by pool_simulator.go, exactly mirroring
	// PonsV2BondingCurve.sellableTokens().
	p.Reserves = entity.PoolReserves{
		quoteReserve.Dec(),
		tokenReserve.Dec(),
	}

	logger.WithFields(logger.Fields{
		"address":      p.Address,
		"quoteReserve": quoteReserve.String(),
		"tokenReserve": tokenReserve.String(),
		"graduated":    graduated,
	}).Info("finished getting new state of pons-v2 curve")

	return p, nil
}
