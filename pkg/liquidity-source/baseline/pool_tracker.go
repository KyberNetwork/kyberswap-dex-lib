package baseline

import (
	"context"
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
	BLV           *uint256.Int `abi:"BLV"`
	Circ          *uint256.Int `abi:"circ"`
	Supply        *uint256.Int `abi:"supply"`
	SwapFee       *uint256.Int `abi:"swapFee"`
	Reserves      *uint256.Int `abi:"reserves"`
	TotalSupply   *uint256.Int `abi:"totalSupply"`
	ConvexityExp  *uint256.Int `abi:"convexityExp"`
	LastInvariant *uint256.Int `abi:"lastInvariant"`
}

type rpcQuoteState struct {
	SnapshotCurveParams     rpcCurveParams `abi:"snapshotCurveParams"`
	QuoteBlockBuyDeltaCirc  *uint256.Int   `abi:"quoteBlockBuyDeltaCirc"`
	QuoteBlockSellDeltaCirc *uint256.Int   `abi:"quoteBlockSellDeltaCirc"`
	TotalSupply             *uint256.Int   `abi:"totalSupply"`
	TotalBTokens            *uint256.Int   `abi:"totalBTokens"`
	TotalReserves           *uint256.Int   `abi:"totalReserves"`
	ReserveDecimals         uint8          `abi:"reserveDecimals"`
	LiquidityFeePct         *uint256.Int   `abi:"liquidityFeePct"`
	PendingSurplus          *uint256.Int   `abi:"pendingSurplus"`
	ShouldSettlePending     bool           `abi:"shouldSettlePendingSurplus"`
	MaxSellDelta            *uint256.Int   `abi:"maxSellDelta"`
	SnapshotActivePrice     *uint256.Int   `abi:"snapshotActivePrice"`
}

type rpcGetQuoteStateResult struct {
	State rpcQuoteState `abi:"state_"`
}

func (s rpcQuoteState) toQuoteState() *QuoteState {
	return &QuoteState{
		SnapshotCurveParams:     s.SnapshotCurveParams.toCurveParams(),
		QuoteBlockBuyDeltaCirc:  cloneOrZeroU256(s.QuoteBlockBuyDeltaCirc),
		QuoteBlockSellDeltaCirc: cloneOrZeroU256(s.QuoteBlockSellDeltaCirc),
		TotalSupply:             cloneOrZeroU256(s.TotalSupply),
		TotalBTokens:            cloneOrZeroU256(s.TotalBTokens),
		TotalReserves:           cloneOrZeroU256(s.TotalReserves),
		ReserveDecimals:         s.ReserveDecimals,
		LiquidityFeePct:         cloneOrZeroU256(s.LiquidityFeePct),
		PendingSurplus:          cloneOrZeroU256(s.PendingSurplus),
		SettlePendingSurplus:    s.ShouldSettlePending,
		MaxSellDelta:            cloneOrZeroU256(s.MaxSellDelta),
		SnapshotActivePrice:     cloneOrZeroU256(s.SnapshotActivePrice),
	}
}

func (p rpcCurveParams) toCurveParams() CurveParams {
	return CurveParams{
		BLV:           cloneOrZeroU256(p.BLV),
		Circ:          cloneOrZeroU256(p.Circ),
		Supply:        cloneOrZeroU256(p.Supply),
		SwapFee:       cloneOrZeroU256(p.SwapFee),
		Reserves:      cloneOrZeroU256(p.Reserves),
		TotalSupply:   cloneOrZeroU256(p.TotalSupply),
		ConvexityExp:  cloneOrZeroU256(p.ConvexityExp),
		LastInvariant: cloneOrZeroU256(p.LastInvariant),
	}
}

func cloneOrZeroU256(x *uint256.Int) *uint256.Int {
	if x == nil {
		return new(uint256.Int)
	}

	return new(uint256.Int).Set(x)
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

	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		logger.WithFields(logger.Fields{
			"poolAddress": p.Address,
			"error":       err,
		}).Errorf("[Baseline] failed to fetch pool state")
		return entity.Pool{}, err
	}
	p.BlockNumber = resp.BlockNumber.Uint64()

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
		cloneOrZeroU256(quoteState.TotalReserves).Dec(),
		cloneOrZeroU256(quoteState.TotalBTokens).Dec(),
	}
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()

	logger.Infof("[Baseline] Finish getting new state of pool: %v", p.Address)
	return p, nil
}
