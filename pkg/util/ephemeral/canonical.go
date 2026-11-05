package ephemeral

import (
	"errors"
	"fmt"
	"math"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/holiman/uint256"
)

var errShort = errors.New("payload ends before its values do")

// checkCanonical checks that payload is exactly the canonical ABI encoding of values of args'
// types: what encoding the values it decodes to gives back, byte for byte. It reads the payload
// as it is, where encoding the decoded values again would cost twice the decoding: every intN and
// uintN within N bits, bools 0 or 1, padding zero, every offset where the encoding puts its value,
// and nothing after the last.
func checkCanonical(args abi.Arguments, payload []byte) error {
	n, err := canonicalSeq(len(args), func(i int) *abi.Type { return &args[i].Type }, payload)
	if err != nil {
		return err
	}
	if n != len(payload) {
		return fmt.Errorf("%d bytes after the values", len(payload)-n)
	}
	return nil
}

// canonicalSeq checks the encoding of count values, of types typ(0) to typ(count-1), at the start
// of data, as a tuple encodes them: a head of static values and offsets, then each dynamic value
// in order. It returns the bytes they take.
func canonicalSeq(count int, typ func(int) *abi.Type, data []byte) (int, error) {
	head := 0
	for i := range count {
		head += headSize(typ(i))
		if head > len(data) {
			return 0, errShort
		}
	}
	pos, end := 0, head
	for i := range count {
		t := typ(i)
		if !isDynamic(t) {
			n, err := canonicalValue(t, data[pos:])
			if err != nil {
				return 0, err
			}
			pos += n
			continue
		}
		off, err := readSize(data[pos:])
		if err != nil {
			return 0, err
		}
		if off != end {
			return 0, fmt.Errorf("offset %d where the value starts at %d", off, end)
		}
		n, err := canonicalValue(t, data[end:])
		if err != nil {
			return 0, err
		}
		pos, end = pos+32, end+n
	}
	return end, nil
}

// canonicalValue checks the encoding of one value of t at the start of data, its offset already
// followed, and returns the bytes it takes.
func canonicalValue(t *abi.Type, data []byte) (int, error) {
	switch t.T {
	case abi.TupleTy:
		return canonicalSeq(len(t.TupleElems), func(i int) *abi.Type { return t.TupleElems[i] }, data)
	case abi.ArrayTy:
		return canonicalSeq(t.Size, func(int) *abi.Type { return t.Elem }, data)
	case abi.SliceTy:
		n, err := readSize(data)
		if err != nil {
			return 0, err
		}
		// Each element takes a word at least, so a length past the data is refused before its loop.
		if n > (len(data)-32)/32 {
			return 0, errShort
		}
		size, err := canonicalSeq(n, func(int) *abi.Type { return t.Elem }, data[32:])
		return 32 + size, err
	case abi.BytesTy, abi.StringTy:
		n, err := readSize(data)
		if err != nil {
			return 0, err
		}
		if n > len(data)-32 {
			return 0, errShort
		}
		padded := (n + 31) / 32 * 32
		if padded > len(data)-32 {
			return 0, errShort
		}
		if !zero(data[32+n : 32+padded]) {
			return 0, errors.New("bytes padding is not zero")
		}
		return 32 + padded, nil
	}

	if len(data) < 32 {
		return 0, errShort
	}
	word := data[:32]
	switch t.T {
	case abi.UintTy:
		var v uint256.Int
		if v.SetBytes32(word).BitLen() > t.Size {
			return 0, fmt.Errorf("value does not fit %s", t)
		}
	case abi.IntTy:
		var v uint256.Int
		if v.SetBytes32(word).Sign() < 0 {
			v.Not(&v)
		}
		if v.BitLen() > t.Size-1 {
			return 0, fmt.Errorf("value does not fit %s", t)
		}
	case abi.BoolTy:
		if !zero(word[:31]) || word[31] > 1 {
			return 0, errors.New("bool is neither 0 nor 1")
		}
	case abi.AddressTy:
		if !zero(word[:12]) {
			return 0, errors.New("address high bytes are not zero")
		}
	case abi.FixedBytesTy:
		if !zero(word[t.Size:]) {
			return 0, fmt.Errorf("%s padding is not zero", t)
		}
	case abi.HashTy:
	default:
		return 0, fmt.Errorf("%s results are not supported", t)
	}
	return 32, nil
}

// headSize is the bytes a value of t takes in the head of the tuple it is in: an offset for a
// dynamic one, the value itself for a static one.
func headSize(t *abi.Type) int {
	if isDynamic(t) {
		return 32
	}
	switch t.T {
	case abi.ArrayTy:
		return t.Size * headSize(t.Elem)
	case abi.TupleTy:
		n := 0
		for _, elem := range t.TupleElems {
			n += headSize(elem)
		}
		return n
	}
	return 32
}

func isDynamic(t *abi.Type) bool {
	switch t.T {
	case abi.BytesTy, abi.StringTy, abi.SliceTy:
		return true
	case abi.ArrayTy:
		return isDynamic(t.Elem)
	case abi.TupleTy:
		for _, elem := range t.TupleElems {
			if isDynamic(elem) {
				return true
			}
		}
	}
	return false
}

// readSize reads an offset or a length: a word whose value fits an int.
func readSize(data []byte) (int, error) {
	if len(data) < 32 {
		return 0, errShort
	}
	var v uint256.Int
	if !v.SetBytes32(data[:32]).IsUint64() || v.Uint64() > math.MaxInt32 {
		return 0, errors.New("offset or length past any payload")
	}
	return int(v.Uint64()), nil
}

func zero(b []byte) bool {
	for _, x := range b {
		if x != 0 {
			return false
		}
	}
	return true
}
