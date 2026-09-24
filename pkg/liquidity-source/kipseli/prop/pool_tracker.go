package prop

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/KyberNetwork/ethrpc"
	"github.com/KyberNetwork/logger"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/ethclient/gethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
	"github.com/goccy/go-json"
	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/entity"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/ladder"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/titan"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool"
	pooltrack "github.com/KyberNetwork/kyberswap-dex-lib/pkg/source/pool/tracker"
	utilabi "github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/abi"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/bignumber"
	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/crypto"
)

const positionCapMethod = "caps"

type PoolTracker struct {
	cfg            *Config
	ethrpcClient   *ethrpc.Client
	signer         *crypto.Eip712Signer
	titanClients   []*rpc.Client
	multicall3Addr common.Address
}

var _ = pooltrack.RegisterFactoryCE0(DexType, NewPoolTracker)

func NewPoolTracker(cfg *Config, ethrpcClient *ethrpc.Client) *PoolTracker {
	return &PoolTracker{
		cfg:            cfg,
		ethrpcClient:   ethrpcClient,
		signer:         crypto.NewEip712Signer(cfg.Quoter[:]),
		titanClients:   titan.NewClients(cfg.Titan),
		multicall3Addr: common.HexToAddress(cfg.Multicall3Address),
	}
}

func (t *PoolTracker) GetNewPoolState(
	ctx context.Context,
	p entity.Pool,
	_ pool.GetNewPoolStateParams,
) (entity.Pool, error) {
	if t.cfg.isPamm() {
		return t.getPammPoolState(ctx, p)
	}
	return t.getPropPoolState(ctx, p)
}

// getPropPoolState probes the EIP-712-signed prop venue: every sample point
// is a signed quote() call batched through ethrpc.
func (t *PoolTracker) getPropPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	tsMs := time.Now().UnixMilli()
	bTsMs := big.NewInt(tsMs)

	tokenAddrs := []common.Address{
		common.HexToAddress(p.Tokens[0].Address),
		common.HexToAddress(p.Tokens[1].Address),
	}

	balances, caps, blockNumber, err := t.fetchBalancesAndCaps(ctx, tokenAddrs)
	if err != nil {
		return p, err
	}
	if len(balances) < 2 || balances[0] == nil || balances[1] == nil {
		return p, ErrInsufficientLiquidity
	}

	maxIn := computeMaxIn(balances, caps)

	req := t.ethrpcClient.NewRequest().SetContext(ctx).SetBlockNumber(blockNumber)

	msgTemplate := DomainType
	msgTemplate.Domain.ChainId = math.NewHexOrDecimal256(int64(t.cfg.ChainID))
	msgTemplate.Domain.VerifyingContract = hexutil.Encode(t.cfg.Verifier[:])

	var pointsPerDir [2][]*big.Int
	var resultsPerDir [2][]*big.Int

	for i := range p.Tokens {
		tokenIn := common.HexToAddress(p.Tokens[i].Address)
		tokenOut := common.HexToAddress(p.Tokens[1-i].Address)

		sig, err := t.signQuoteMsg(msgTemplate, tokenIn, tokenOut, bTsMs)
		if err != nil {
			return p, err
		}

		points := ladder.SamplePoints(p, i, sampleBound(balances[i], maxIn[i]), balances[1-i])
		pointsPerDir[i] = points

		// Output must be **big.Int (&results[j]), not *big.Int: go-ethereum's
		// abi.Copy treats a *big.Int destination's Elem() (a big.Int struct)
		// as a wrapper struct and writes into its field 0 (an unrelated bool),
		// producing "cannot unmarshal *big.Int in to bool" instead of the
		// decoded amount.
		results := make([]*big.Int, len(points))
		for j, pt := range points {
			req.AddCall(&ethrpc.Call{
				ABI:    swapABI,
				Target: t.cfg.RouterAddress,
				Method: "quote",
				Params: []any{tokenIn, pt, tokenOut, bTsMs, sig},
			}, []any{&results[j]})
		}
		resultsPerDir[i] = results
	}

	if _, err := req.TryAggregate(); err != nil {
		if _, err := req.SetBlockNumber(nil).TryAggregate(); err != nil {
			return p, err
		}
	}

	var ladders [2][]ladder.Point
	for dir := range p.Tokens {
		t.applyBuffer(resultsPerDir[dir])
		ladders[dir] = ladder.CollectLadder(pointsPerDir[dir], resultsPerDir[dir])
	}

	extra := Extra{Ladders: ladders}
	extraBytes, err := json.Marshal(extra)
	if err != nil {
		return p, err
	}

	p.Extra = string(extraBytes)
	p.Reserves = entity.PoolReserves{balances[0].String(), balances[1].String()}
	p.Timestamp = time.Now().Unix()
	p.BlockNumber = blockNumber.Uint64()

	return p, nil
}

func (t *PoolTracker) signQuoteMsg(template apitypes.TypedData, tokenIn, tokenOut common.Address, bTsMs *big.Int) ([]byte, error) {
	msg := template
	msg.Message = apitypes.TypedDataMessage{
		"tokenIn":            [20]byte(tokenIn),
		"tokenOut":           [20]byte(tokenOut),
		"timestampInMilisec": bTsMs,
	}
	return t.signer.Sign(msg)
}

func (t *PoolTracker) applyBuffer(results []*big.Int) {
	if t.cfg.Buffer <= 0 {
		return
	}
	buf := big.NewInt(t.cfg.Buffer)
	for _, r := range results {
		if r == nil {
			continue
		}
		r.Mul(r, buf)
		r.Div(r, bignumber.BasisPoint)
	}
}

func (t *PoolTracker) fetchBalancesAndCaps(ctx context.Context, tokenAddrs []common.Address) ([]*big.Int, []*big.Int, *big.Int, error) {
	var balances, caps []*big.Int
	req := t.ethrpcClient.NewRequest().SetContext(ctx)
	req.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: t.cfg.LensAddress,
		Method: "getReserveBalances",
		Params: []any{tokenAddrs},
	}, []any{&balances})
	req.AddCall(&ethrpc.Call{
		ABI:    lensABI,
		Target: t.cfg.LensAddress,
		Method: "getReserveBalanceCap",
		Params: []any{tokenAddrs},
	}, []any{&caps})

	res, err := req.TryBlockAndAggregate()
	if err != nil {
		return nil, nil, nil, err
	}
	return balances, caps, res.BlockNumber, nil
}

// getPammPoolState probes the Titan-quoted pAMM venue: every sample point is
// a plain Lens.quote() call batched through Multicall3, with whatever Titan
// state override is currently available applied to the whole batch.
func (t *PoolTracker) getPammPoolState(ctx context.Context, p entity.Pool) (entity.Pool, error) {
	var staticExtra StaticExtra
	if err := json.Unmarshal([]byte(p.StaticExtra), &staticExtra); err != nil {
		return p, err
	}

	routerAddr := common.HexToAddress(staticExtra.RouterAddress)
	token0Addr := common.HexToAddress(p.Tokens[0].Address)
	token1Addr := common.HexToAddress(p.Tokens[1].Address)

	titanCh := lo.Async(func() titan.State {
		return titan.FetchState(ctx, t.titanClients, t.cfg.Titan.Timeout)
	})

	// Fresh vault address; setSwapImpl() can change it any time.
	var vaultAddr common.Address
	if _, err := t.ethrpcClient.NewRequest().SetContext(ctx).
		AddCall(&ethrpc.Call{
			ABI:    routerPammABI,
			Target: routerAddr.Hex(),
			Method: "wallet",
		}, []any{&vaultAddr}).
		TryAggregate(); err != nil {
		return p, err
	}

	var bal0, bal1, cap0, cap1 *big.Int
	req := t.ethrpcClient.NewRequest().SetContext(ctx).
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
		}, []any{&bal1})
	if t.cfg.PositionCapAddress != "" {
		req.AddCall(&ethrpc.Call{
			ABI:    positionCapABI,
			Target: t.cfg.PositionCapAddress,
			Method: positionCapMethod,
			Params: []any{token0Addr},
		}, []any{&cap0}).
			AddCall(&ethrpc.Call{
				ABI:    positionCapABI,
				Target: t.cfg.PositionCapAddress,
				Method: positionCapMethod,
				Params: []any{token1Addr},
			}, []any{&cap1})
	}
	resp, err := req.TryBlockAndAggregate()
	if err != nil {
		return p, err
	}
	blockNumber := resp.BlockNumber

	titanState := <-titanCh

	ladders, err := t.probePammQuotes(ctx, p, [2]common.Address{token0Addr, token1Addr}, [2]*big.Int{bal0, bal1}, [2]*big.Int{cap0, cap1}, blockNumber, titanState)
	if err != nil {
		return p, err
	}

	if allEmpty(ladders) {
		logger.WithFields(logger.Fields{"dexId": t.cfg.DexID, "pool": p.Address}).
			Warn("all quotes returned 0 (price stale), skipping pool update")
		return p, nil
	}

	extra := Extra{Ladders: ladders, SO: titanState.ToStateOverrides()}
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

func (t *PoolTracker) probePammQuotes(
	ctx context.Context,
	p entity.Pool,
	tokens [2]common.Address,
	balances [2]*big.Int,
	caps [2]*big.Int,
	blockNumber *big.Int,
	titanState titan.State,
) ([2][]ladder.Point, error) {
	type mcall struct {
		Target   common.Address `json:"target"`
		CallData []byte         `json:"callData"`
	}
	type mresult struct {
		Success    bool   `json:"success"`
		ReturnData []byte `json:"returnData"`
	}

	maxIn := computeMaxIn(balances[:], caps[:])
	quoteTarget := common.HexToAddress(t.cfg.LensAddress)

	var pointsPerDir [2][]*big.Int
	var calls []mcall
	var refDir []int

	for dir := range 2 {
		points := ladder.SamplePoints(p, dir, sampleBound(balances[dir], maxIn[dir]), balances[1-dir])
		pointsPerDir[dir] = points

		tokenIn, tokenOut := tokens[dir], tokens[1-dir]
		for _, amt := range points {
			cd, err := lensPammABI.Pack("quote", tokenIn, amt, tokenOut)
			if err != nil {
				return [2][]ladder.Point{}, err
			}
			calls = append(calls, mcall{Target: quoteTarget, CallData: cd})
			refDir = append(refDir, dir)
		}
	}

	mcCalldata, err := multicall3ABI.Pack("tryAggregate", false, calls)
	if err != nil {
		return [2][]ladder.Point{}, err
	}

	override := make(map[common.Address]gethclient.OverrideAccount, len(titanState.Overrides))
	for addr, acc := range titanState.Overrides {
		override[addr] = acc
	}

	gc := gethclient.New(t.ethrpcClient.GetETHClient().Client())
	callMsg := ethereum.CallMsg{To: &t.multicall3Addr, Data: mcCalldata}
	var raw []byte
	if titanState.BlockTimestamp != 0 {
		raw, err = gc.CallContractWithBlockOverrides(ctx, callMsg, blockNumber, &override,
			gethclient.BlockOverrides{Time: titanState.BlockTimestamp})
	} else {
		raw, err = gc.CallContract(ctx, callMsg, blockNumber, &override)
	}
	if err != nil {
		return [2][]ladder.Point{}, err
	}

	var results []mresult
	if err := multicall3ABI.UnpackIntoInterface(&results, "tryAggregate", raw); err != nil {
		return [2][]ladder.Point{}, err
	}
	if len(results) != len(refDir) {
		return [2][]ladder.Point{}, fmt.Errorf("multicall3 result count mismatch: got %d want %d", len(results), len(refDir))
	}

	var resultsPerDir [2][]*big.Int
	resultsPerDir[0] = make([]*big.Int, 0, len(pointsPerDir[0]))
	resultsPerDir[1] = make([]*big.Int, 0, len(pointsPerDir[1]))
	for i, dir := range refDir {
		amtOut := big.NewInt(0)
		if results[i].Success && len(results[i].ReturnData) >= 32 {
			amtOut = new(big.Int).SetBytes(results[i].ReturnData[len(results[i].ReturnData)-32:])
		}
		resultsPerDir[dir] = append(resultsPerDir[dir], amtOut)
	}

	var ladders [2][]ladder.Point
	for dir := range 2 {
		ladders[dir] = ladder.CollectLadder(pointsPerDir[dir], resultsPerDir[dir])
	}

	return ladders, nil
}

// sampleBound picks the tighter of the venue's own balance and its published
// cap room as the grid's ceiling — a token with no cap published falls back
// to its balance.
func sampleBound(balance, maxIn *big.Int) *big.Int {
	bound := balance
	if maxIn != nil && maxIn.Sign() > 0 && maxIn.Cmp(bound) < 0 {
		bound = maxIn
	}
	return bound
}

// computeMaxIn reports how much of each token the venue can still absorb
// before its published cap. A token with no cap published (nil, zero, or
// max-uint) gets no entry and keeps falling back to its raw balance.
func computeMaxIn(balances, caps []*big.Int) []*big.Int {
	maxIn := make([]*big.Int, len(caps))
	for i, c := range caps {
		if c == nil || c.Sign() <= 0 || c.Cmp(bignumber.MaxUint256) == 0 {
			continue
		}
		if i < len(balances) && balances[i] != nil && c.Cmp(balances[i]) > 0 {
			maxIn[i] = new(big.Int).Sub(c, balances[i])
		}
	}
	return maxIn
}

func allEmpty(ladders [2][]ladder.Point) bool {
	return len(ladders[0]) == 0 && len(ladders[1]) == 0
}

func balStr(v *big.Int) string {
	if v == nil {
		return "0"
	}
	return v.String()
}
