package pamm

import (
	"context"
	"fmt"
	"math/big"
	"sort"
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
	cfg            *Config
	ethrpcClient   *ethrpc.Client
	titanClients   []*rpc.Client
	multicall3Addr common.Address
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:            cfg,
		ethrpcClient:   ethrpcClient,
		titanClients:   newTitanClients(cfg.Titan),
		multicall3Addr: common.HexToAddress(cfg.Multicall3Address),
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	routerAddr := common.HexToAddress(staticExtra.RouterAddress)
	token0Addr := common.HexToAddress(p.Tokens[0].Address)
	token1Addr := common.HexToAddress(p.Tokens[1].Address)

	overridesCh := lo.Async(func() titanPammState {
		return t.fetchStateOverrides(ctx)
	})

	// Fresh vault address; setSwapImpl() can change it any time.
	var vaultAddr common.Address
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    routerABI,
			Target: routerAddr.Hex(),
			Method: "wallet",
		}, []any{&vaultAddr}).
		TryAggregate(); err != nil {
		return p, err
	}

	var bal0, bal1 *big.Int
	resp, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    utilabi.Erc20ABI,
			Target: token0Addr.Hex(),
			Method: utilabi.Erc20BalanceOfMethod,
			Params: []any{vaultAddr},
		}, []any{&bal0}).
		AddCall(&ethrpc.Call{
			ABI:    utilabi.Erc20ABI,
			Target: token1Addr.Hex(),
			Method: utilabi.Erc20BalanceOfMethod,
			Params: []any{vaultAddr},
		}, []any{&bal1}).
		TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	blockNumber := resp.BlockNumber

	titanState := <-overridesCh

	samples, err := t.fetchQuotes(ctx, p, token0Addr, token1Addr, bal0, bal1, blockNumber, titanState)
	if err != nil {
		return p, err
	}

	if allZero(samples) {
		logger.WithFields(logger.Fields{"dexId": t.cfg.DexID, "pool": p.Address}).
			Warn("all quotes returned 0 (price stale), skipping pool update")
		return p, nil
	}

	warnGapInQuotes(p, samples)

	extra := Extra{
		Samples:        filterAllSamples(samples, bal0, bal1),
		SO:             titanOverridesToMap(titanState.Overrides),
		BlockTimestamp: titanState.BlockTimestamp,
	}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{balStr(bal0), balStr(bal1)}
	p.BlockNumber = blockNumber.Uint64()
	p.Timestamp = time.Now().Unix()
	return p, nil
}

// fetchQuotes probes the PAMM quote target at every sample point in a single
// Multicall3 eth_call, with Titan pricing overrides applied to the whole batch.
// The PUR-aware target requires block.timestamp to equal the priority update
// timestamp, so Titan's timestamp is forwarded as a block override when present.
func (t *PoolTracker) fetchQuotes(
	ctx context.Context,
	p entity.Pool,
	token0, token1 common.Address,
	bal0, bal1 *big.Int,
	blockNumber *big.Int,
	titanState titanPammState,
) ([][][2]*big.Int, error) {
	type mcall struct {
		Target   common.Address `json:"target"`
		CallData []byte         `json:"callData"`
	}
	type mresult struct {
		Success    bool   `json:"success"`
		ReturnData []byte `json:"returnData"`
	}

	quoteTarget := common.HexToAddress(t.cfg.LensAddress)
	tokens := [2]common.Address{token0, token1}
	reserves := [2]*big.Int{bal0, bal1}

	type sampleRef struct {
		dir   int
		amtIn *big.Int
	}
	var refs []sampleRef
	var calls []mcall

	for dir := range 2 {
		tokenIn := tokens[dir]
		tokenOut := tokens[1-dir]
		for _, amtIn := range buildSamplePoints(int(p.Tokens[dir].Decimals), reserves[dir]) {
			cd, err := lensABI.Pack("quote", tokenIn, amtIn, tokenOut)
			if err != nil {
				return nil, err
			}
			calls = append(calls, mcall{Target: quoteTarget, CallData: cd})
			refs = append(refs, sampleRef{dir: dir, amtIn: amtIn})
		}
	}

	mcCalldata, err := multicall3ABI.Pack("tryAggregate", false, calls)
	if err != nil {
		return nil, err
	}

	override := make(map[common.Address]gethclient.OverrideAccount, len(titanState.Overrides))
	for addr, acc := range titanState.Overrides {
		override[addr] = acc
	}

	gc := gethclient.New(t.ethrpcClient.GetETHClient().Client())
	callMsg := ethereum.CallMsg{To: &t.multicall3Addr, Data: mcCalldata}
	var raw []byte
	if titanState.BlockTimestamp != 0 {
		raw, err = gc.CallContractWithBlockOverrides(
			ctx,
			callMsg,
			blockNumber,
			&override,
			gethclient.BlockOverrides{Time: titanState.BlockTimestamp},
		)
	} else {
		raw, err = gc.CallContract(ctx, callMsg, blockNumber, &override)
	}
	if err != nil {
		return nil, err
	}

	var results []mresult
	if err := multicall3ABI.UnpackIntoInterface(&results, "tryAggregate", raw); err != nil {
		return nil, err
	}
	if len(results) != len(refs) {
		return nil, fmt.Errorf("multicall3 result count mismatch: got %d want %d", len(results), len(refs))
	}

	samples := make([][][2]*big.Int, 2)
	for dir := range 2 {
		samples[dir] = make([][2]*big.Int, 0, len(refs))
	}

	for i, ref := range refs {
		res := results[i]
		outReserve := reserves[1-ref.dir]

		if !res.Success {
			samples[ref.dir] = append(samples[ref.dir], [2]*big.Int{new(big.Int).Set(ref.amtIn), big.NewInt(0)})
			continue
		}

		amtOut := big.NewInt(0)
		if len(res.ReturnData) >= 32 {
			amtOut = new(big.Int).SetBytes(res.ReturnData[len(res.ReturnData)-32:])
		}
		if outReserve != nil && outReserve.Sign() > 0 && amtOut.Cmp(outReserve) >= 0 {
			amtOut = big.NewInt(0)
		}
		samples[ref.dir] = append(samples[ref.dir], [2]*big.Int{new(big.Int).Set(ref.amtIn), amtOut})
	}

	return samples, nil
}

// buildSamplePoints: 10^k + 3·10^k levels around tokenDecimals, plus reserve
// fractions near capacity. Sorted, deduped.
func buildSamplePoints(tokenDecimals int, reserve *big.Int) []*big.Int {
	points := make([]*big.Int, 0, 2*sampleSize+len(maxInSampleBps))

	start := max(0, tokenDecimals-sampleSize/2)
	for i, k := 0, start; i < sampleSize; i, k = i+1, k+1 {
		points = append(points, bignumber.TenPowInt(k))
		points = append(points, new(big.Int).Mul(bignumber.TenPowInt(k), big.NewInt(3)))
	}

	if reserve != nil && reserve.Sign() > 0 {
		for _, bps := range maxInSampleBps {
			pt := new(big.Int).Mul(reserve, big.NewInt(int64(bps)))
			pt.Div(pt, bignumber.BasisPoint)
			if pt.Sign() > 0 {
				points = append(points, pt)
			}
		}
	}

	sort.Slice(points, func(a, b int) bool { return points[a].Cmp(points[b]) < 0 })
	return dedupSorted(points)
}

func filterAllSamples(samples [][][2]*big.Int, bal0, bal1 *big.Int) [][][2]*big.Int {
	reserves := [2]*big.Int{bal0, bal1}
	out := make([][][2]*big.Int, 2)
	for dir := range 2 {
		out[dir] = filterSamples(samples[dir], reserves[1-dir])
	}
	return out
}

func filterSamples(samples [][2]*big.Int, outputReserve *big.Int) [][2]*big.Int {
	valid := samples[:0]
	for _, s := range samples {
		if s[0] == nil || s[1] == nil || s[1].Sign() <= 0 {
			continue
		}
		if outputReserve != nil && outputReserve.Sign() > 0 && s[1].Cmp(outputReserve) >= 0 {
			continue
		}
		valid = append(valid, s)
	}
	return valid
}

func allZero(samples [][][2]*big.Int) bool {
	for _, dir := range samples {
		for _, s := range dir {
			if s[1] != nil && s[1].Sign() > 0 {
				return false
			}
		}
	}
	return true
}

// warnGapInQuotes logs when a direction has positive → 0 → positive — usually
// indicates a per-sample revert mid-range that may mask real liquidity.
func warnGapInQuotes(p entity.Pool, samples [][][2]*big.Int) {
	for dir := range samples {
		seenPositive := false
		zeroRunStart := -1
		for i := range samples[dir] {
			pt := samples[dir][i]
			if pt[0] == nil || pt[1] == nil {
				continue
			}
			if pt[1].Sign() > 0 {
				if zeroRunStart >= 0 {
					logger.WithFields(logger.Fields{
						"pool":           p.Address,
						"dir":            dir,
						"holeFromAmount": samples[dir][zeroRunStart][0].String(),
						"holeToAmount":   samples[dir][i-1][0].String(),
						"resumeAmount":   pt[0].String(),
					}).Warn("quote gap detected (positive -> zero -> positive)")
					zeroRunStart = -1
				}
				seenPositive = true
				continue
			}
			if seenPositive && zeroRunStart < 0 {
				zeroRunStart = i
			}
		}
	}
}

func dedupSorted(sorted []*big.Int) []*big.Int {
	if len(sorted) <= 1 {
		return sorted
	}
	result := sorted[:1]
	for _, v := range sorted[1:] {
		if v.Cmp(result[len(result)-1]) != 0 {
			result = append(result, v)
		}
	}
	return result
}

func balStr(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}
