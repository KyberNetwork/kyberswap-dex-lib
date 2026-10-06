package flywheelfun

import (
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func abiType(name string, fields []abi.ArgumentMarshaling) abi.Type {
	t, e := abi.NewType(name, "", fields)
	if e != nil {
		panic(e)
	}
	return t
}

var keyFields = []abi.ArgumentMarshaling{{Name: "currency0", Type: "address"}, {Name: "currency1", Type: "address"}, {Name: "fee", Type: "uint24"}, {Name: "tickSpacing", Type: "int24"}, {Name: "hooks", Type: "address"}}
var keyArguments = abi.Arguments{{Type: abiType("address", nil)}, {Type: abiType("address", nil)}, {Type: abiType("uint24", nil)}, {Type: abiType("int24", nil)}, {Type: abiType("address", nil)}}
var routeArguments = abi.Arguments{{Type: abiType("tuple[]", []abi.ArgumentMarshaling{{Name: "kind", Type: "uint8"}, {Name: "pool", Type: "address"}, {Name: "key", Type: "tuple", Components: keyFields}})}}
var tradeArguments = abi.Arguments{{Type: abiType("tuple", []abi.ArgumentMarshaling{{Name: "token", Type: "address"}, {Name: "minQuote", Type: "uint256"}, {Name: "minOutput", Type: "uint256"}, {Name: "deadline", Type: "uint256"}, {Name: "route", Type: "bytes"}, {Name: "minRefundETH", Type: "uint256"}, {Name: "refundRoute", Type: "bytes"}})}}

type AdapterTrade struct {
	Token        common.Address
	MinQuote     *big.Int
	MinOutput    *big.Int
	Deadline     *big.Int
	Route        []byte
	MinRefundETH *big.Int
	RefundRoute  []byte
}
type encodedHop struct {
	Kind uint8
	Pool common.Address
	Key  PoolKey
}

// EncodeRoute returns the settlement route bytes for hops; reverse walks quote -> WETH (sells, refunds).
func EncodeRoute(hops []RouteHop, reverse bool) ([]byte, error) {
	if len(hops) == 0 {
		return nil, nil
	}
	if len(hops) > 3 {
		return nil, ErrState
	}
	values := make([]encodedHop, len(hops))
	for i, h := range hops {
		if _, err := h.id(); err != nil {
			return nil, err
		}
		k := h.Key
		if h.Kind == 3 {
			k = PoolKey{Fee: big.NewInt(0), TickSpacing: big.NewInt(0)}
		}
		values[i] = encodedHop{h.Kind, h.Pool, k}
	}
	if reverse {
		slices.Reverse(values)
	}
	data, err := routeArguments.Pack(values)
	if err != nil {
		return nil, err
	}
	return append([]byte{0x4e, 0x41, 0x54, 0x31}, data...), nil
}

// EncodeTradeData returns the abi.encode(Trade) payload for the execution module.
// The Kyber executor's outer transaction envelope is intentionally separate.
// Refund protection uses the reverse swap's ETH output, not the larger refund
// that also includes returned platform fees.
func EncodeTradeData(info SwapInfo, slippageBps uint16, now, deadline uint64) ([]byte, error) {
	if slippageBps >= 10000 || deadline <= now || deadline-now > 3600 || !common.IsHexAddress(info.Token) || common.HexToAddress(info.Token) == (common.Address{}) || info.MinQuote == nil || info.MinQuote.IsZero() || info.AmountOut == nil || info.AmountOut.IsZero() || info.Refund == nil {
		return nil, ErrAmount
	}
	minimum := func(x *big.Int) *big.Int {
		v := new(big.Int).Mul(x, big.NewInt(int64(10000-slippageBps)))
		v.Div(v, big.NewInt(10000))
		if v.Sign() == 0 {
			v.SetInt64(1)
		}
		return v
	}
	t := AdapterTrade{Token: common.HexToAddress(info.Token), MinQuote: minimum(info.MinQuote.ToBig()), MinOutput: minimum(info.AmountOut.ToBig()), Deadline: new(big.Int).SetUint64(deadline), MinRefundETH: new(big.Int)}
	var err error
	t.Route, err = EncodeRoute(info.Route, !info.Buy)
	if err != nil {
		return nil, err
	}
	if !info.Refund.IsZero() {
		if !info.Buy || info.RefundRouteOutput == nil || info.RefundRouteOutput.IsZero() {
			return nil, ErrAmount
		}
		t.MinRefundETH = minimum(info.RefundRouteOutput.ToBig())
		t.RefundRoute, err = EncodeRoute(info.Route, true)
		if err != nil {
			return nil, err
		}
	}
	return tradeArguments.Pack(t)
}
