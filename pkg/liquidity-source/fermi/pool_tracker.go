package fermi

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	config       *Config
	ethrpcClient *ethrpc.Client
	titanClients []*rpc.Client
}

var _ = pooltrack.RegisterFactoryCE(DexType, NewPoolTracker)

func NewPoolTracker(
	config *Config,
	ethrpcClient *ethrpc.Client,
) (*PoolTracker, error) {
	return &PoolTracker{
		config:       config,
		ethrpcClient: ethrpcClient,
		titanClients: newTitanClients(config.Titan),
	}, nil
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	startTime := time.Now()
	logger.WithFields(logger.Fields{
		"dex_id":       t.config.DexId,
		"pool_address": p.Address,
	}).Info("started getting new pool state")

	p, err := t.getPoolState(ctx, p)
	if err != nil {
		return p, err
	}

	logger.WithFields(logger.Fields{
		"dex_id":       t.config.DexId,
		"pool_address": p.Address,
		"duration_ms":  time.Since(startTime).Milliseconds(),
	}).Info("finished getting new pool state")

	return p, nil
}

func (t *PoolTracker) getPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, fmt.Errorf("unmarshal staticExtra: %w", err)
	}

	t0Addr := common.HexToAddress(p.Tokens[0].Address)
	t1Addr := common.HexToAddress(p.Tokens[1].Address)

	overridesCh := lo.Async(func() map[common.Address]gethclient.OverrideAccount {
		return t.fetchStateOverrides(ctx)
	})

	var (
		bal0, bal1         *big.Int
		fermi, traderVault common.Address
	)
	_, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    fermiSwapperABI,
			Target: t.config.FermiSwapper,
			Method: methodFermi,
		}, []any{&fermi}).
		AddCall(&ethrpc.Call{
			ABI:    fermiSwapperABI,
			Target: t.config.FermiSwapper,
			Method: methodTraderVault,
		}, []any{&traderVault}).
		TryBlockAndAggregate()
	if err != nil {
		return p, fmt.Errorf("fetch fermi and vault balances: %w", err)
	}

	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    utilabi.Erc20ABI,
			Target: t0Addr.Hex(),
			Method: utilabi.Erc20BalanceOfMethod,
			Params: []any{traderVault},
		}, []any{&bal0}).
		AddCall(&ethrpc.Call{
			ABI:    utilabi.Erc20ABI,
			Target: t1Addr.Hex(),
			Method: utilabi.Erc20BalanceOfMethod,
			Params: []any{traderVault},
		}, []any{&bal1}).
		TryBlockAndAggregate()
	if err != nil {
		return p, fmt.Errorf("fetch vault balances: %w", err)
	}
	blockNumber := resp.BlockNumber.Uint64()

	overrides := <-overridesCh
	midPrice := t.extractMidPrice(t0Addr, t1Addr, fermi, overrides)

	// Persist the FULL Titan override (all pairs' slots) into every pool's Extra.
	// A multi-hop route through two Fermi pools needs both pairs' price +
	// lastUpdatedBlock at the SAME block. Storing the whole snapshot per-pool
	// means the aggregator-encoding merge produces a consistent state_overrides
	// blob with matching lastUpdatedBlock values across every Fermi hop.
	extra := Extra{
		Fermi:       fermi.String(),
		TraderVault: traderVault.String(),
		BlockNumber: blockNumber,
	}
	if so := toStateOverrides(overrides, fermi); len(so) > 0 {
		extra.StateOverrides = &so
	}

	// fetchCurveParams uses gethclient.CallContract when overrides are present.
	// The response includes a 32-byte ABI tuple offset prefix that the
	// auto-decoder cannot handle, so we call directly and strip it.
	raw, curveErr := t.fetchCurveParams(ctx, t0Addr, t1Addr, fermi, overrides)
	if curveErr != nil {
		logger.WithFields(logger.Fields{
			"error": curveErr.Error(),
			"t0":    t0Addr.Hex(),
			"t1":    t1Addr.Hex(),
		}).Warn("fermi: getPairParams failed")
	} else {
		curve := &CurveData{
			FeeBaseBps:         raw.FeeBaseBps,
			SafetyFeeBps:       raw.SafetyFeeBps,
			ScalingDenominator: bigToString(raw.ScalingDenominator),
			MaxAmountIn:        bigToString(raw.MaxAmountIn),
			SizeSpline:         knotsToType(raw.SizeSpline),
			InventorySpline:    knotsToType(raw.InvSpline),
			VaultReserve0:      bigToString(bal0),
			VaultReserve1:      bigToString(bal1),
		}
		if midPrice != nil {
			curve.MidPrice = midPrice.String()
			extra.Curve = curve
		} else {
			logger.WithFields(logger.Fields{
				"dex_id":       t.config.DexId,
				"pool_address": p.Address,
			}).Warn("fermi: midPrice not available from Titan, skipping curve")
		}
	}

	t0Dec, t1Dec := p.Tokens[0].Decimals, p.Tokens[1].Decimals

	if extra.Curve != nil && extra.Curve.TokenInDecScale == "" {
		ds0 := new(big.Int).Exp(bignumber.Ten, big.NewInt(int64(t0Dec)), nil)
		ds1 := new(big.Int).Exp(bignumber.Ten, big.NewInt(int64(t1Dec)), nil)
		extra.Curve.TokenInDecScale = ds0.String()
		extra.Curve.TokenOutDecScale = ds1.String()
	}

	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}
	staticExtraBytes, err := json.Marshal(staticExtra)
	if err != nil {
		return p, err
	}

	r0, r1 := "0", "0"
	if extra.Curve != nil {
		if extra.Curve.VaultReserve0 != "" {
			r0 = extra.Curve.VaultReserve0
		}
		if extra.Curve.VaultReserve1 != "" {
			r1 = extra.Curve.VaultReserve1
		}
	}

	p.Extra = string(extraBytes)
	p.StaticExtra = string(staticExtraBytes)
	p.Reserves = entity.PoolReserves{r0, r1}
	p.BlockNumber = blockNumber
	p.Timestamp = time.Now().Unix()

	return p, nil
}

// fetchCurveParams calls FermiEngine.getPairParams via eth_call, optionally
// with state overrides. We bypass ethrpc's auto-decoder because getPairParams
// returns an ABI response with a 32-byte tuple offset prefix.
func (t *PoolTracker) fetchCurveParams(
	ctx context.Context,
	t0, t1, fermi common.Address,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*engineCurveResult, error) {
	data, err := fermiEngineABI.Pack(methodGetPairParams, t0, t1)
	if err != nil {
		return nil, fmt.Errorf("pack getPairParams: %w", err)
	}

	var resp []byte
	if len(overrides) > 0 {
		gc := gethclient.New(t.ethrpcClient.GetETHClient().Client())
		resp, err = gc.CallContract(ctx, ethereum.CallMsg{To: &fermi, Data: data}, nil, &overrides)
	} else {
		resp, err = t.ethrpcClient.GetETHClient().CallContract(ctx, ethereum.CallMsg{To: &fermi, Data: data}, nil)
	}
	if err != nil {
		return nil, fmt.Errorf("callContract getPairParams: %w", err)
	}

	// Strip the 32-byte tuple offset prefix when present.
	if len(resp) >= 64 {
		firstWord := new(big.Int).SetBytes(resp[:32]).Uint64()
		if firstWord == 32 {
			resp = resp[32:]
		}
	}

	vals, err := fermiEngineABI.Unpack(methodGetPairParams, resp)
	if err != nil {
		return nil, fmt.Errorf("unpack getPairParams: %w", err)
	}
	if len(vals) < 6 {
		return nil, fmt.Errorf("getPairParams: expected 6 return values, got %d", len(vals))
	}

	toKnots := func(v any) ([]engineKnotABI, error) {
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		var ks []engineKnotABI
		if err := json.Unmarshal(b, &ks); err != nil {
			return nil, err
		}
		return ks, nil
	}

	sizeSpline, err := toKnots(vals[4])
	if err != nil {
		return nil, fmt.Errorf("decode sizeSpline: %w", err)
	}
	invSpline, err := toKnots(vals[5])
	if err != nil {
		return nil, fmt.Errorf("decode invSpline: %w", err)
	}

	return &engineCurveResult{
		FeeBaseBps:         vals[0].(uint16),
		SafetyFeeBps:       vals[1].(uint16),
		ScalingDenominator: vals[2].(*big.Int),
		MaxAmountIn:        vals[3].(*big.Int),
		SizeSpline:         sizeSpline,
		InvSpline:          invSpline,
	}, nil
}

// extractMidPrice reads midPrice from state overrides for the given token pair.
func (t *PoolTracker) extractMidPrice(
	t0, t1, fermi common.Address,
	overrides map[common.Address]gethclient.OverrideAccount,
) *big.Int {
	price, _, _ := t.findPairOverride(t0, t1, fermi, overrides)
	return price
}

// findPairOverride locates this pair's PairState in the Titan override set.
func (t *PoolTracker) findPairOverride(
	t0, t1, fermi common.Address,
	overrides map[common.Address]gethclient.OverrideAccount,
) (*big.Int, common.Hash, bool) {
	if len(overrides) == 0 {
		return nil, common.Hash{}, false
	}
	acct, ok := overrides[fermi]
	if !ok || len(acct.StateDiff) == 0 {
		return nil, common.Hash{}, false
	}

	fwd, rev := pairKeyForTokens(t0, t1)
	for _, pk := range []common.Hash{fwd, rev} {
		base := pairBaseSlot(pk)
		priceSlot := slotOffset(base, 1)
		if word, found := acct.StateDiff[priceSlot]; found {
			return decodeMidPrice(word), base, true
		}
	}
	return nil, common.Hash{}, false
}

// toStateOverrides serializes the Titan override set into the JSON-friendly
// shape stored in Extra.StateOverrides. The full snapshot (every slot Titan
// returned) is persisted per-pool so multi-hop routes have consistent
// lastUpdatedBlock values across all Fermi pairs touched.
func toStateOverrides(
	overrides map[common.Address]gethclient.OverrideAccount,
	fermi common.Address,
) StateOverrides {
	if len(overrides) == 0 {
		return nil
	}
	out := make(StateOverrides, len(overrides))
	for addr, acct := range overrides {
		if addr != fermi || len(acct.StateDiff) == 0 {
			continue
		}
		diff := make(map[string]string, len(acct.StateDiff))
		for slot, val := range acct.StateDiff {
			diff[slot.Hex()] = val.Hex()
		}
		out[strings.ToLower(addr.Hex())] = diff
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

type engineKnotABI struct {
	XLo *big.Int `json:"xLo"`
	XHi *big.Int `json:"xHi"`
	C0  *big.Int `json:"c0"`
	C1  *big.Int `json:"c1"`
	C2  *big.Int `json:"c2"`
	C3  *big.Int `json:"c3"`
}

type engineCurveResult struct {
	FeeBaseBps         uint16          `abi:"feeBaseBps"`
	SafetyFeeBps       uint16          `abi:"safetyFeeBps"`
	ScalingDenominator *big.Int        `abi:"scalingDenominator"`
	MaxAmountIn        *big.Int        `abi:"maxAmountIn"`
	SizeSpline         []engineKnotABI `abi:"sizeSpline"`
	InvSpline          []engineKnotABI `abi:"invSpline"`
}

func bigToString(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}

func knotsToType(in []engineKnotABI) []Knot {
	out := make([]Knot, len(in))
	for i, k := range in {
		out[i] = Knot{
			XLo: bigToString(k.XLo),
			XHi: bigToString(k.XHi),
			C0:  bigToString(k.C0),
			C1:  bigToString(k.C1),
			C2:  bigToString(k.C2),
			C3:  bigToString(k.C3),
		}
	}
	return out
}
