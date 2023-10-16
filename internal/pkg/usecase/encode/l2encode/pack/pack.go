package pack

import (
	"fmt"
	"math/big"
	"reflect"

	"github.com/ethereum/go-ethereum/common"
)

type Int24 int32
type UInt24 uint32
type UInt96 *big.Int
type UInt160 *big.Int
type UInt200 *big.Int

// RawBytes is used when you want to simply concat the bytes into the current packing data.
// Useful when encoding data with nested structs inside.
type RawBytes []byte

var _AddressType = reflect.TypeOf(common.BytesToAddress(nil))
var _Int24Type = reflect.TypeOf(Int24(0))
var _UInt24Type = reflect.TypeOf(UInt24(0))
var _UInt96Type = reflect.TypeOf(UInt96(big.NewInt(0)))
var _UInt160Type = reflect.TypeOf(UInt160(big.NewInt(0)))
var _UInt200Type = reflect.TypeOf(UInt200(big.NewInt(0)))
var _RawByteType = reflect.TypeOf(RawBytes([]byte{}))
var _BigIntType = reflect.TypeOf(big.NewInt(0))
var _BoolType = reflect.TypeOf(false)
var _Int8Type = reflect.TypeOf(int8(0))
var _Int16Type = reflect.TypeOf(int16(0))
var _Int32Type = reflect.TypeOf(int32(0))
var _Int64Type = reflect.TypeOf(int64(0))
var _UInt8Type = reflect.TypeOf(uint8(0))
var _UInt16Type = reflect.TypeOf(uint16(0))
var _UInt32Type = reflect.TypeOf(uint32(0))
var _UInt64Type = reflect.TypeOf(uint64(0))

func Pack(args ...interface{}) ([]byte, error) {
	var ret []byte

	for _, arg := range args {
		rt := reflect.TypeOf(arg)

		// Special handler for slice, which recursion call
		// PackMultiple for inner elements
		if rt.Kind() == reflect.Slice {
			rv := reflect.ValueOf(arg)
			iArg := make([]interface{}, rv.Len())
			for i := 0; i < rv.Len(); i++ {
				iArg[i] = rv.Index(i).Interface()
			}
			var _ret []byte
			var err error
			if rt == _RawByteType {
				_ret, err = Pack(iArg...)
			} else if rt.Elem() == _UInt8Type {
				_ret = PackSliceBytes(arg.([]byte))
			} else {
				_ret, err = PackSlice(iArg)
			}

			if err != nil {
				return nil, err
			}
			ret = append(ret, _ret...)
			continue
		}

		switch rt {
		case _Int24Type:
			ret = append(ret, PackInt24(arg.(Int24))...)
		case _UInt24Type:
			ret = append(ret, PackInt(uint32(arg.(UInt24)), 3)...) // Compressed to 3 bytes
		case _UInt96Type:
			ret = append(ret, PackBigInt(arg.(UInt96), 12)...) // Compressed to 12 bytes
		case _UInt160Type:
			ret = append(ret, PackBigInt(arg.(UInt160), 20)...) // Compressed to 20 bytes
		case _UInt200Type:
			ret = append(ret, PackBigInt(arg.(UInt200), 25)...) // compressed to 25 bytes
		case _BigIntType:
			ret = append(ret, PackBigInt(arg.(*big.Int), 16)...) // Compressed to 16 bytes
		case _AddressType:
			ret = append(ret, PackAddress(arg.(common.Address))...)
		case _BoolType:
			ret = append(ret, PackBoolean(arg.(bool))...)

		case _Int8Type:
			ret = append(ret, PackInt(arg.(int8), 1)...)
		case _Int16Type:
			ret = append(ret, PackInt(arg.(int16), 2)...)
		case _Int32Type:
			ret = append(ret, PackInt(arg.(int32), 4)...)
		case _Int64Type:
			ret = append(ret, PackInt(arg.(int64), 8)...)

		case _UInt8Type:
			ret = append(ret, PackInt(arg.(uint8), 1)...)
		case _UInt16Type:
			ret = append(ret, PackInt(arg.(uint16), 2)...)
		case _UInt32Type:
			ret = append(ret, PackInt(arg.(uint32), 4)...)
		case _UInt64Type:
			ret = append(ret, PackInt(arg.(uint64), 8)...)

		default:
			return nil, fmt.Errorf("could not pack element, unknown type: %v", rt)
		}
	}
	return ret, nil
}

func PackAddress(arg common.Address) []byte {
	return arg.Bytes()
}

func PackBoolean(arg bool) []byte {
	if arg {
		return []byte{1}
	}
	return []byte{0}
}

func PackBigInt(arg *big.Int, size uint8) []byte {
	ret := make([]byte, size)
	return arg.FillBytes(ret)
}

// PackInt24 uses two's complement method
// to present a number in binary
func PackInt24(arg Int24) []byte {
	argInt32 := int32(arg)
	if argInt32 < 0 {
		argInt32 = -argInt32
		argInt32 ^= (1 << 24) - 1
		argInt32 += 1
	}
	return PackInt(argInt32, 3) // Compressed to 3 bytes
}

// PackInt stores []byte in big-endian system (stores the most significant byte at the smallest memory address).
func PackInt[T uint8 | uint16 | uint32 | uint64 | uint | int8 | int16 | int32 | int64 | int](arg T, size uint8) []byte {
	ret := make([]byte, size)
	var i uint8
	for i = 0; i < size; i++ {
		ret[i] = byte(arg >> (8 * (size - i - 1)))
	}
	return ret
}

// PackSlice packs the length of slice into the first 1 byte, then pack the following inner elements.
func PackSlice(arg []interface{}) ([]byte, error) {
	packed, err := Pack(arg...)
	if err != nil {
		return nil, err
	}
	return append(PackInt(len(arg), 1), packed...), nil
}

// PackSliceBytes packs the length of byte into the first 4 bytes, then pack the following inner byte elements.
func PackSliceBytes(arg []byte) []byte {
	return append(PackInt(len(arg), 4), arg...)
}
