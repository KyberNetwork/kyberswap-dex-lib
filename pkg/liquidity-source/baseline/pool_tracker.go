package baseline

import (
	"context"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
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

func NewPoolTracker(
	cfg *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       cfg,
		ethrpcClient: ethrpcClient,
	}, nil
}

type rpcCurveParams struct {
	BLV           *big.Int `abi:"BLV"`
	Circ          *big.Int `abi:"circ"`
	Supply        *big.Int `abi:"supply"`
	SwapFee       *big.Int `abi:"swapFee"`
	Reserves      *big.Int `abi:"reserves"`
	TotalSupply   *big.Int `abi:"totalSupply"`
	ConvexityExp  *big.Int `abi:"convexityExp"`
	LastInvariant *big.Int `abi:"lastInvariant"`
}

type rpcQuoteState struct {
	SnapshotCurveParams     rpcCurveParams `abi:"snapshotCurveParams"`
	QuoteBlockBuyDeltaCirc  *big.Int       `abi:"quoteBlockBuyDeltaCirc"`
	QuoteBlockSellDeltaCirc *big.Int       `abi:"quoteBlockSellDeltaCirc"`
	TotalSupply             *big.Int       `abi:"totalSupply"`
	TotalBTokens            *big.Int       `abi:"totalBTokens"`
	TotalReserves           *big.Int       `abi:"totalReserves"`
	ReserveDecimals         uint8          `abi:"reserveDecimals"`
	LiquidityFeePct         *big.Int       `abi:"liquidityFeePct"`
	PendingSurplus          *big.Int       `abi:"pendingSurplus"`
	ShouldSettlePending     bool           `abi:"shouldSettlePendingSurplus"`
	MaxSellDelta            *big.Int       `abi:"maxSellDelta"`
	SnapshotActivePrice     *big.Int       `abi:"snapshotActivePrice"`
}

type rpcGetQuoteStateResult struct {
	State rpcQuoteState `abi:"state_"`
}

func (s rpcQuoteState) toQuoteState() *QuoteState {
	return &QuoteState{
		SnapshotCurveParams:     s.SnapshotCurveParams.toCurveParams(),
		QuoteBlockBuyDeltaCirc:  uint256.MustFromBig(nonNilBI(s.QuoteBlockBuyDeltaCirc)),
		QuoteBlockSellDeltaCirc: uint256.MustFromBig(nonNilBI(s.QuoteBlockSellDeltaCirc)),
		TotalSupply:             uint256.MustFromBig(nonNilBI(s.TotalSupply)),
		TotalBTokens:            uint256.MustFromBig(nonNilBI(s.TotalBTokens)),
		TotalReserves:           uint256.MustFromBig(nonNilBI(s.TotalReserves)),
		ReserveDecimals:         s.ReserveDecimals,
		LiquidityFeePct:         uint256.MustFromBig(nonNilBI(s.LiquidityFeePct)),
		PendingSurplus:          uint256.MustFromBig(nonNilBI(s.PendingSurplus)),
		SettlePendingSurplus:    s.ShouldSettlePending,
		MaxSellDelta:            uint256.MustFromBig(nonNilBI(s.MaxSellDelta)),
		SnapshotActivePrice:     uint256.MustFromBig(nonNilBI(s.SnapshotActivePrice)),
	}
}

func (p rpcCurveParams) toCurveParams() CurveParams {
	return CurveParams{
		BLV:           uint256.MustFromBig(nonNilBI(p.BLV)),
		Circ:          uint256.MustFromBig(nonNilBI(p.Circ)),
		Supply:        uint256.MustFromBig(nonNilBI(p.Supply)),
		SwapFee:       uint256.MustFromBig(nonNilBI(p.SwapFee)),
		Reserves:      uint256.MustFromBig(nonNilBI(p.Reserves)),
		TotalSupply:   uint256.MustFromBig(nonNilBI(p.TotalSupply)),
		ConvexityExp:  uint256.MustFromBig(nonNilBI(p.ConvexityExp)),
		LastInvariant: uint256.MustFromBig(nonNilBI(p.LastInvariant)),
	}
}

func (d *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	logger.Infof("[Baseline] Start getting new state of pool: %v", p.Address)

	if len(p.Tokens) != 2 {
		return entity.Pool{}, ErrPoolNotFound
	}

	// Pool address is the bToken address
	bTokenAddr := common.HexToAddress(p.Address)

	var result rpcGetQuoteStateResult

	req := d.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    relayABI,
		Target: d.config.RelayAddress,
		Method: methodGetQuoteState,
		Params: []any{bTokenAddr},
	}, []any{&result})

	if _, err := req.TryAggregate(); err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Baseline] failed to fetch pool state")
		return entity.Pool{}, err
	}

	extra := Extra{
		RelayAddress: d.config.RelayAddress,
	}
	quoteState := result.State
	extra.QuoteState = quoteState.toQuoteState()

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return entity.Pool{}, err
	}

	p.Reserves = entity.PoolReserves{
		nonNilBI(quoteState.TotalReserves).String(),
		nonNilBI(quoteState.TotalBTokens).String(),
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.Infof("[Baseline] Finish getting new state of pool: %v", p.Address)
	return p, nil
}
