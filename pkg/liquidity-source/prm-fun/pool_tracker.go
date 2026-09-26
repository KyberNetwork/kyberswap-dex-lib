package prmfun

import (
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
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
	return &PoolTracker{config: config, ethrpcClient: ethrpcClient}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, nil)
}

func (t *PoolTracker) GetNewPoolStateWithOverrides(
	ctx context.Context,
	p entity.Pool,
	params pool.GetNewPoolStateWithOverridesParams,
) (entity.Pool, error) {
	return t.getNewPoolState(ctx, p, params.Overrides)
}

// getNewPoolState re-reads phase, virtualMeme/virtualDesk (via getReserves - virtualMeme()/
// virtualDesk() aren't in the vendor's ABI individually), memeSold, and deskRaised in one
// multicall. GraduationDesk, the curve address, and the meme token are immutable (set once
// at discovery in StaticExtra) and are never re-read here.
func (t *PoolTracker) getNewPoolState(
	ctx context.Context,
	p entity.Pool,
	overrides map[common.Address]gethclient.OverrideAccount,
) (entity.Pool, error) {
	logger.WithFields(logger.Fields{"address": p.Address}).Info("start getting new state of pool")

	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	var (
		phase          uint8
		reservesResult GetReservesResult
		memeSold       *big.Int
		deskRaised     *big.Int
		pair           common.Address
		native         bool
	)

	// Multicall's block.number is an L1 parent on Robinhood. The RPC block
	// identifies the actual L2 state and must also be returned to the encoder.
	head, err := t.ethrpcClient.GetBlockNumber(ctx)
	if err != nil {
		return p, err
	}
	if head == 0 {
		return p, ErrInvalidState
	}
	_, err = t.ethrpcClient.NewRequest().SetOverrides(overrides).SetContext(ctx).SetBlockNumber(new(big.Int).SetUint64(head)).
		AddCall(&ethrpc.Call{
			ABI: memeCurveABI, Target: staticExtra.CurveAddress, Method: memeCurveMethodPhase,
		}, []any{&phase}).
		AddCall(&ethrpc.Call{
			ABI: memeCurveABI, Target: staticExtra.CurveAddress, Method: memeCurveMethodGetReserves,
		}, []any{&reservesResult}).
		AddCall(&ethrpc.Call{
			ABI: memeCurveABI, Target: staticExtra.CurveAddress, Method: memeCurveMethodMemeSold,
		}, []any{&memeSold}).
		AddCall(&ethrpc.Call{
			ABI: memeCurveABI, Target: staticExtra.CurveAddress, Method: memeCurveMethodDeskRaised,
		}, []any{&deskRaised}).
		AddCall(&ethrpc.Call{ABI: memeCurveABI, Target: staticExtra.CurveAddress, Method: "deskToken"}, []any{&pair}).
		AddCall(&ethrpc.Call{ABI: memeCurveABI, Target: staticExtra.CurveAddress, Method: "isNativeQuote"}, []any{&native}).
		Aggregate()
	if err != nil {
		return p, err
	}
	if phase > PhasePaused {
		return p, ErrInvalidState
	}
	for _, amount := range []*big.Int{reservesResult.QuoteReserve, reservesResult.TokenReserve, memeSold, deskRaised} {
		if amount == nil || amount.Sign() < 0 || amount.BitLen() > 256 {
			return p, ErrInvalidState
		}
	}
	if len(p.Tokens) != 2 || p.Tokens[0] == nil || !strings.EqualFold(p.Tokens[0].Address, hexutil.Encode(pair[:])) {
		return p, ErrInvalidToken
	}
	// Refresh the native flag also for persisted ETH-only rows from the previous
	// implementation, which did not include it in StaticExtra.
	staticExtra.IsNativeQuote = native
	staticBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return p, err
	}
	virtualDesk := uint256.MustFromBig(reservesResult.QuoteReserve)
	virtualMeme := uint256.MustFromBig(reservesResult.TokenReserve)
	uMemeSold := uint256.MustFromBig(memeSold)
	uDeskRaised := uint256.MustFromBig(deskRaised)

	extra := Extra{
		Phase:       phase,
		VirtualMeme: virtualMeme,
		VirtualDesk: virtualDesk,
		MemeSold:    uMemeSold,
		DeskRaised:  uDeskRaised,
	}
	target, err := uint256.FromDecimal(staticExtra.GraduationDesk)
	if err != nil || !validState(extra, target) {
		return p, ErrInvalidState
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	p.StaticExtra = string(staticBytes)
	p.Extra = string(extraBytes)
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = head

	if phase != PhaseTrading {
		p.Reserves = entity.PoolReserves{"0", "0"}
		if phase == PhaseGraduated {
			p.Timestamp = 1
		}
		logger.WithFields(logger.Fields{"address": p.Address, "phase": phase}).
			Info("prm-fun pool is not trading")
		return p, nil
	}

	p.Reserves = entity.PoolReserves{virtualDesk.Dec(), virtualMeme.Dec()}

	logger.WithFields(logger.Fields{"address": p.Address}).Info("finish getting new state of pool")
	return p, nil
}
