package inverse

import (
	"context"
	_ "embed"
	"math/big"
	"strings"

	"github.com/KyberNetwork/ethrpc"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/goccy/go-json"

	uniswapv4 "github.com/KyberNetwork/kyberswap-dex-lib/pkg/liquidity-source/uniswap/v4"
)

//go:embed abi.json
var abiJSON string
var contractABI = func() abi.ABI {
	a, e := abi.JSON(strings.NewReader(abiJSON))
	if e != nil {
		panic(e)
	}
	return a
}()
var managerAddress = common.HexToAddress("0x8366a39cc670b4001a1121b8f6a443a643e40951")
var quoteAddress = common.HexToAddress("0x0bd7d308f8e1639fab988df18a8011f41eacad73")
var controllerAddress = common.HexToAddress("0x6d0009504d129cf5002dba61d9ae8575aa79314c")

const stateViewAddress = "0xf3334192d15450cdd385c8b70e03f9a6bd9e673b"

type slot0Response struct{ SqrtPriceX96, Tick, ProtocolFee, LpFee *big.Int }
type positionResponse struct{ Liquidity, FeeGrowthInside0LastX128, FeeGrowthInside1LastX128 *big.Int }
type growthResponse struct{ FeeGrowthInside0X128, FeeGrowthInside1X128 *big.Int }

// Track batches independent calls and pins both dependent batches to the exact
// block selected by the v4 tracker. Overrides are forwarded to BOTH batches.
func (h *Hook) Track(ctx context.Context, p *uniswapv4.HookParam) (json.RawMessage, error) {
	if p == nil || p.RpcClient == nil || p.Pool == nil || len(p.Pool.Tokens) != 2 || p.BlockNumber == nil || !p.BlockNumber.IsUint64() || p.BlockNumber.Sign() <= 0 || p.Cfg == nil || p.Cfg.ChainID != 4663 {
		return nil, ErrState
	}
	var static uniswapv4.StaticExtra
	if err := json.Unmarshal([]byte(p.Pool.StaticExtra), &static); err != nil {
		return nil, err
	}
	if static.Fee != 3000 || static.TickSpacing != 60 || static.HooksAddress != p.HookAddress || static.IsNative[0] || static.IsNative[1] || p.HookAddress == (common.Address{}) {
		return nil, ErrState
	}
	for _, t := range p.Pool.Tokens {
		if t == nil || t.Decimals != 18 {
			return nil, ErrState
		}
	}
	poolID := common.HexToHash(p.Pool.Address)
	stateView := stateViewAddress
	if p.Cfg.StateViewAddress != "" {
		stateView = p.Cfg.StateViewAddress
	}
	req := p.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(p.BlockNumber).SetOverrides(p.Overrides)
	call := func(target common.Address, method string, args []any, result any) {
		req.AddCall(&ethrpc.Call{ABI: contractABI, Target: hexutil.Encode(target[:]), Method: method, Params: args}, []any{result})
	}
	var token, quoteToken, manager, controller common.Address
	var id common.Hash
	var initialized, closed, guard bool
	var phase uint8
	var slot slot0Response
	var position positionResponse
	var growth growthResponse
	var activeLiquidity *big.Int
	// Decode uint256 getters to big.Int: go-ethereum's ABI does not decode
	// directly to uint256.Int. Convert only after the successful aggregate.
	names := []string{"initialShares", "initialQuote", "reserveShares", "reserveQuote", "nativeQuote", "roundingQuote", "sequence", "liquidity"}
	values := make([]*big.Int, len(names))
	for i, name := range names {
		call(p.HookAddress, name, nil, &values[i])
	}
	call(p.HookAddress, "token", nil, &token)
	call(p.HookAddress, "quoteToken", nil, &quoteToken)
	call(p.HookAddress, "poolManager", nil, &manager)
	call(p.HookAddress, "poolId", nil, &id)
	call(p.HookAddress, "initialized", nil, &initialized)
	call(p.HookAddress, "closed", nil, &closed)
	call(p.HookAddress, "phase", nil, &phase)
	call(p.HookAddress, "prepaymentGuard", nil, &guard)
	view := common.HexToAddress(stateView)
	call(view, "getSlot0", []any{poolID}, &slot)
	call(view, "getLiquidity", []any{poolID}, &activeLiquidity)
	call(view, "getPositionInfo", []any{poolID, p.HookAddress, n(minTick), n(maxTick), common.Hash{}}, &position)
	call(view, "getFeeGrowthInside", []any{poolID, n(minTick), n(maxTick)}, &growth)
	call(managerAddress, "protocolFeeController", nil, &controller)
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	inverse0 := common.HexToAddress(p.Pool.Tokens[0].Address) == token
	t0, t1 := token, quoteToken
	if !inverse0 {
		t0, t1 = t1, t0
	}
	if id != poolID || manager != managerAddress || quoteToken != quoteAddress || token == quoteToken || token == (common.Address{}) ||
		common.HexToAddress(p.Pool.Tokens[0].Address) != t0 || common.HexToAddress(p.Pool.Tokens[1].Address) != t1 ||
		strings.Compare(hexutil.Encode(t0[:]), hexutil.Encode(t1[:])) >= 0 || slot.LpFee.Uint64() != 3000 || phase != 0 || guard ||
		activeLiquidity.Cmp(position.Liquidity) != 0 || activeLiquidity.Cmp(values[7]) != 0 {
		return nil, ErrState
	}
	req = p.RpcClient.NewRequest().SetContext(ctx).SetBlockNumber(p.BlockNumber).SetOverrides(p.Overrides)
	tokenNames := []string{"indexRay", "workingIndexRay", "custodiedShares", "nativeBalance", "totalShares", "prepaidNominal", "prepaidShares"}
	tv := make([]*big.Int, len(tokenNames))
	for i, name := range tokenNames {
		call(token, name, nil, &tv[i])
	}
	var market, tokenManager common.Address
	var tokenPhase uint8
	var hookQuote, accrued *big.Int
	call(token, "market", nil, &market)
	call(token, "poolManager", nil, &tokenManager)
	call(token, "phase", nil, &tokenPhase)
	call(quoteToken, "balanceOf", []any{p.HookAddress}, &hookQuote)
	call(manager, "protocolFeesAccrued", []any{token}, &accrued)
	if _, err := req.Aggregate(); err != nil {
		return nil, err
	}
	if market != p.HookAddress || tokenManager != manager || tokenPhase != 0 || tv[0].Cmp(tv[1]) != 0 || tv[4].Cmp(values[0]) != 0 || tv[5].Sign() != 0 || tv[6].Sign() != 0 || accrued.Sign() != 0 {
		return nil, ErrState
	}
	// Fee-growth subtraction is intentionally modulo 2^256, as in Position.update.
	growth0 := u(growth.FeeGrowthInside0X128)
	growth0.Sub(growth0, u(position.FeeGrowthInside0LastX128))
	growth1 := u(growth.FeeGrowthInside1X128)
	growth1.Sub(growth1, u(position.FeeGrowthInside1LastX128))
	s := Extra{Version: 1, Live: initialized && !closed, Inverse0: inverse0, FeeControllerSupported: controller == controllerAddress, ProtocolFee: uint32(slot.ProtocolFee.Uint64()), BlockNumber: p.BlockNumber.Uint64(),
		InitialShares: *u(values[0]), InitialQuote: *u(values[1]), ReserveShares: *u(values[2]), ReserveQuote: *u(values[3]), NativeQuote: *u(values[4]), RoundingQuote: *u(values[5]), Sequence: *u(values[6]), Liquidity: *u(values[7]),
		Index: *u(tv[0]), CustodiedShares: *u(tv[2]), NativeInverse: *u(tv[3]), HookQuote: *u(hookQuote), SqrtPriceX96: *u(slot.SqrtPriceX96), Tick: int(slot.Tick.Int64()),
		Fees0: *u(md(growth0.ToBig(), position.Liquidity, q128)), Fees1: *u(md(growth1.ToBig(), position.Liquidity, q128))}
	if s.Live {
		if err := s.validate(); err != nil {
			return nil, err
		}
	}
	raw, err := json.Marshal(&s)
	if err != nil {
		return nil, err
	}
	h.State = s
	p.Pool.BlockNumber = s.BlockNumber
	return raw, nil
}
