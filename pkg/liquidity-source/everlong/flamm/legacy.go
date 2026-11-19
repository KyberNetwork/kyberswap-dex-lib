package everlongflamm

import (
	"bytes"
	"context"
	"math/big"
	"strings"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/goccy/go-json"
	"github.com/holiman/uint256"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
)

// A legacy pool is a two-token FLAMM of the first Factory build: no tokenCount/tokenConfig/tradablePairs, the pool
// asset and its loan asset, and `previewSwap(poolAssetIn, amountIn)`. It is listed with StaticExtra.Legacy and
// tracked by probing that quote on a size grid, one ladder per direction; the simulator and the encoder are shared.
const legacyPoolABIJSON = `[
{"type":"function","name":"asset","inputs":[],"outputs":[{"name":"","type":"address"}],"stateMutability":"view"},
{"type":"function","name":"loanAsset","inputs":[],"outputs":[{"name":"","type":"address"}],"stateMutability":"view"},
{"type":"function","name":"paused","inputs":[],"outputs":[{"name":"","type":"bool"}],"stateMutability":"view"},
{"type":"function","name":"previewSwap","inputs":[{"name":"poolAssetIn","type":"bool"},{"name":"amountIn","type":"uint256"}],
 "outputs":[{"name":"amountInUsed","type":"uint256"},{"name":"amountOut","type":"uint256"},{"name":"feeWad","type":"uint256"}],
 "stateMutability":"view"}]`

var legacyPoolABI = func() abi.ABI {
	a, err := abi.JSON(bytes.NewReader([]byte(legacyPoolABIJSON)))
	if err != nil {
		panic(err)
	}
	return a
}()

// Gas of a legacy pool's `swap`: the integer average of the pool's call in 7 sells and 6 buys, 5 sells and 4 buys
// mined on an anvil fork of Base (sizes 1e3 to 5e7 sats; 1 to 10,000 USDC) and the rest from Tenderly simulations of
// routed fills, 2026-11-18.
const (
	legacyGasSell int64 = 876_408
	legacyGasBuy  int64 = 836_386
)

type legacyPreview struct {
	AmountInUsed *big.Int `abi:"amountInUsed"`
	AmountOut    *big.Int `abi:"amountOut"`
	FeeWad       *big.Int `abi:"feeWad"`
}

// legacyTokens reads a legacy pool's asset and loan asset; an error where the pool is not one.
func (u *PoolsListUpdater) legacyTokens(ctx context.Context, pool string) ([]*entity.PoolToken, error) {
	var asset, loan common.Address
	if _, err := u.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{ABI: legacyPoolABI, Target: pool, Method: "asset"}, []any{&asset}).
		AddCall(&ethrpc.Call{ABI: legacyPoolABI, Target: pool, Method: "loanAsset"}, []any{&loan}).
		Aggregate(); err != nil {
		return nil, err
	}
	if asset == (common.Address{}) || loan == (common.Address{}) || asset == loan {
		return nil, ErrInvalidToken
	}
	return []*entity.PoolToken{{Address: strings.ToLower(asset.Hex()), Swappable: true},
		{Address: strings.ToLower(loan.Hex()), Swappable: true}}, nil
}

// legacyGrid is the probe sizes of a token of `decimals`: 1, 2 and 5 times each power of ten from 10^(decimals-3)
// to 10^(decimals+6). A size the pool refuses (price band, minimum) drops out of the ladder.
func legacyGrid(decimals uint8) []*big.Int {
	if decimals == 0 {
		decimals = 6
	}
	var grid []*big.Int
	for e := max(int(decimals)-3, 0); e <= int(decimals)+6; e++ {
		for _, m := range []int64{1, 2, 5} {
			grid = append(grid, new(big.Int).Mul(big.NewInt(m), new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(e)), nil)))
		}
	}
	return grid
}

// legacyStaticExtra marks a listed entity's StaticExtra as a legacy pool's.
func legacyStaticExtra(staticExtra string) string {
	var se StaticExtra
	_ = json.Unmarshal([]byte(staticExtra), &se)
	se.Legacy = true
	b, _ := json.Marshal(se)
	return string(b)
}

func (t *PoolTracker) trackLegacy(ctx context.Context, p entity.Pool, se StaticExtra,
	overrides map[common.Address]gethclient.OverrideAccount) (entity.Pool, error) {
	if len(p.Tokens) != 2 {
		return p, ErrInvalidState
	}
	var isPool, paused bool
	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetOverrides(overrides).
		AddCall(&ethrpc.Call{ABI: factoryABI, Target: se.Factory, Method: "isPool", Params: []any{common.HexToAddress(p.Address)}},
			[]any{&isPool}).
		AddCall(&ethrpc.Call{ABI: legacyPoolABI, Target: p.Address, Method: "paused"}, []any{&paused})
	grids := [2][]*big.Int{legacyGrid(p.Tokens[0].Decimals), legacyGrid(p.Tokens[1].Decimals)}
	var probes [2][]legacyPreview
	for d := range probes {
		probes[d] = make([]legacyPreview, len(grids[d]))
		for k, a := range grids[d] {
			req.AddCall(&ethrpc.Call{ABI: legacyPoolABI, Target: p.Address, Method: "previewSwap",
				Params: []any{d == 0, a}}, []any{&probes[d][k]})
		}
	}
	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	if res.BlockNumber == nil || len(res.Result) != 2+len(grids[0])+len(grids[1]) || !res.Result[0] || !res.Result[1] {
		return p, ErrRead
	}
	if !isPool || paused {
		return dark(p, res.BlockNumber), nil
	}

	var ladders []Ladder
	at := 2
	for d := range probes {
		gas := [2]int64{legacyGasSell, legacyGasBuy}[d]
		l := Ladder{In: uint8(d), Out: uint8(1 - d), Cells: []Cell{{Gas: gas}}, Second: gas}
		var prevIn, prevOut uint256.Int
		for k, pr := range probes[d] {
			if !res.Result[at+k] || pr.AmountInUsed == nil || pr.AmountOut == nil {
				continue
			}
			var used, out uint256.Int
			if used.SetFromBig(pr.AmountInUsed) || out.SetFromBig(pr.AmountOut) || used.IsZero() || out.IsZero() ||
				used.BitLen() > maxPointBits || out.BitLen() > maxPointBits || !used.Gt(&prevIn) || out.Lt(&prevOut) {
				continue
			}
			prevIn.Set(&used)
			prevOut.Set(&out)
			l.Points = append(l.Points, Point{new(uint256.Int).Set(&used), new(uint256.Int).Set(&out)})
			if pr.AmountInUsed.Cmp(grids[d][k]) < 0 { // the pool clipped the input: its ceiling
				l.Capped = true
				break
			}
		}
		at += len(grids[d])
		if len(l.Points) != 0 {
			ladders = append(ladders, l)
		}
	}

	now := time.Now().Unix()
	e := Extra{Ladders: ladders, HaircutBps: t.cfg.haircutBps()}
	if len(ladders) != 0 {
		e.LiveAt = now
	}
	extra, err := json.Marshal(e)
	if err != nil {
		return p, err
	}
	p.Extra = string(extra)
	p.Reserves = rooms(2, ladders)
	p.BlockNumber = res.BlockNumber.Uint64()
	p.Timestamp = now
	return p, nil
}
