package flywheelfun

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
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

// encodeRoute returns NativeQuoteRoute's mixed-route bytes for a one-hop route. One hop
// encodes the same in both directions, so buys, sells and refunds share it.
func encodeRoute(h routeHop) ([]byte, error) {
	data, err := routeArguments.Pack([]routeHop{h})
	if err != nil {
		return nil, err
	}
	return append([]byte{0x4e, 0x41, 0x54, 0x31}, data...), nil
}
