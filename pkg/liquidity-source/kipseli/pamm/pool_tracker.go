package pamm

import (
	"context"
	"encoding/json"
	"math/big"
	"sort"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
)

type PoolTracker struct {
	cfg          *Config
	ethrpcClient *ethrpc.Client
	titanClients []*rpc.Client
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:          cfg,
		ethrpcClient: ethrpcClient,
		titanClients: newTitanClients(cfg.Titan),
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

	overridesCh := lo.Async(func() map[common.Address]gethclient.OverrideAccount {
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

	titanOverrides := <-overridesCh

	samples, err := t.fetchQuotes(ctx, p, routerAddr, token0Addr, token1Addr, bal0, bal1, blockNumber, titanOverrides)
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
		Samples:          filterAllSamples(samples, bal0, bal1),
		SO:               titanOverridesToMap(titanOverrides),
		LastUpdatedBlock: blockNumber.Uint64(),
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

// fetchQuotes probes Router.swap() at each sample point via eth_call with state
// overrides (Titan pricing + tokenIn.balanceOf[router] = amtIn).
func (t *PoolTracker) fetchQuotes(
	ctx context.Context,
	p entity.Pool,
	routerAddr common.Address,
	token0, token1 common.Address,
	bal0, bal1 *big.Int,
	blockNumber *big.Int,
	titanOverrides map[common.Address]gethclient.OverrideAccount,
) ([][][2]*big.Int, error) {
	gc := gethclient.New(t.ethrpcClient.GetETHClient().Client())

	samples := make([][][2]*big.Int, 2)
	tokens := [2]common.Address{token0, token1}
	reserves := [2]*big.Int{bal0, bal1}

	for dir := range 2 {
		tokenIn := tokens[dir]
		tokenOut := tokens[1-dir]
		outReserve := reserves[1-dir]

		points := buildSamplePoints(int(p.Tokens[dir].Decimals), reserves[dir])
		samples[dir] = make([][2]*big.Int, 0, len(points))

		for _, amtIn := range points {
			override := mergeOverrides(titanOverrides, tokenIn, routerAddr, amtIn)

			callData, err := routerABI.Pack("swap", tokenIn, amtIn, tokenOut, routerAddr)
			if err != nil {
				return nil, err
			}

			raw, err := gc.CallContract(ctx, ethereum.CallMsg{To: &routerAddr, Data: callData}, blockNumber, &override)
			if err != nil {
				// Single-sample revert is non-fatal; record 0 so filterSamples drops it.
				samples[dir] = append(samples[dir], [2]*big.Int{new(big.Int).Set(amtIn), big.NewInt(0)})
				continue
			}

			amtOut := big.NewInt(0)
			if len(raw) >= 32 {
				amtOut = new(big.Int).SetBytes(raw[len(raw)-32:])
			}
			if outReserve != nil && outReserve.Sign() > 0 && amtOut.Cmp(outReserve) >= 0 {
				amtOut = big.NewInt(0)
			}

			samples[dir] = append(samples[dir], [2]*big.Int{new(big.Int).Set(amtIn), amtOut})
		}
	}

	return samples, nil
}

// mergeOverrides combines Titan pricing overrides with the per-sample
// tokenIn.balanceOf[router] = amtIn override. Balance slot wins on conflict.
func mergeOverrides(
	titan map[common.Address]gethclient.OverrideAccount,
	tokenIn, routerAddr common.Address,
	amtIn *big.Int,
) map[common.Address]gethclient.OverrideAccount {
	balSlot := balanceOfSlot(routerAddr, tokenIn)
	balVal := common.BigToHash(amtIn)

	out := make(map[common.Address]gethclient.OverrideAccount, len(titan)+1)
	for addr, acc := range titan {
		if addr != tokenIn {
			out[addr] = acc
		}
	}

	acc := gethclient.OverrideAccount{StateDiff: map[common.Hash]common.Hash{balSlot: balVal}}
	if existing, ok := titan[tokenIn]; ok {
		for slot, val := range existing.StateDiff {
			if slot != balSlot {
				acc.StateDiff[slot] = val
			}
		}
		acc.Balance = existing.Balance
		acc.Nonce = existing.Nonce
		acc.Code = existing.Code
	}
	out[tokenIn] = acc
	return out
}

// balanceOfSlot computes keccak256(account || mappingSlot) for ERC-20 balances.
func balanceOfSlot(account common.Address, token common.Address) common.Hash {
	var buf [64]byte
	copy(buf[12:32], account.Bytes())
	buf[63] = knownBalanceOfSlot(token)
	return common.BytesToHash(crypto.Keccak256(buf[:]))
}

// Confirmed slots: WETH=3, USDC=9. Unknown tokens fall through to slot 0,
// which yields amtOut=0 and is dropped by filterSamples.
func knownBalanceOfSlot(token common.Address) byte {
	switch strings.ToLower(token.Hex()) {
	case "0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2": // WETH
		return 3
	case "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48": // USDC
		return 9
	default:
		return 0
	}
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
