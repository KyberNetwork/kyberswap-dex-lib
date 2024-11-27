package testutil

import (
	"fmt"
	"reflect"
	"regexp"

	"github.com/KyberNetwork/int256"
	uniswapcoreentities "github.com/daoleno/uniswap-sdk-core/entities"
	"github.com/davecgh/go-spew/spew"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/holiman/uint256"
)

// CmpOpts prepare []cmp.Option to compare
//
// - unexported fields
//
// - empty fields
//
// - zero/nil *uint256.Int
//
// - zero/nil *int256.Int
//
// - compare uniswapcoreentities.{Token,Native,Ether,BaseCurrency} via [reflect.DeepEqual] to bypass their Equal() method
func CmpOpts(_ ...any) []cmp.Option {
	return []cmp.Option{
		cmp.Exporter(func(t reflect.Type) bool { return true }),
		cmpopts.EquateEmpty(),
		cmp.Comparer(func(lhs, rhs *uint256.Int) bool {
			if lhs == nil {
				return rhs == nil || rhs.IsZero()
			}
			if rhs == nil {
				return lhs.IsZero()
			}
			return lhs.Cmp(rhs) == 0
		}),
		cmp.Comparer(func(lhs, rhs *int256.Int) bool {
			if lhs == nil {
				return rhs == nil || rhs.IsZero()
			}
			if rhs == nil {
				return lhs.IsZero()
			}
			return lhs.Cmp(rhs) == 0
		}),
		cmp.Comparer(func(lhs, rhs *uniswapcoreentities.Token) bool {
			return reflect.DeepEqual(lhs, rhs)
		}),
		cmp.Comparer(func(lhs, rhs *uniswapcoreentities.Native) bool {
			return reflect.DeepEqual(lhs, rhs)
		}),
		cmp.Comparer(func(lhs, rhs *uniswapcoreentities.Ether) bool {
			return reflect.DeepEqual(lhs, rhs)
		}),
		cmp.Comparer(func(lhs, rhs *uniswapcoreentities.BaseCurrency) bool {
			return reflect.DeepEqual(lhs, rhs)
		}),
	}
}

var (
	ptrPattern       = regexp.MustCompile(`(?m)\(0x\w{10}\)`)
	bigIntPtrPattern = regexp.MustCompile(`(?m)\(\*big\.Int\)\(0x\w{10}\)`)
	dumpConfig       = spew.ConfigState{
		Indent:            " ",
		SortKeys:          true,
		DisableCapacities: true, // we don't need to compare slice/map capacity
	}
)

// DumpWithNormalizedPointers spew.Sdump(v) where pointer values are normalized from zero.
// It is useful to check if two structs have the same relationshop of their inner pointers.
func DumpWithNormalizedPointers(v any) string {
	dump := []byte(dumpConfig.Sdump(v))

	// ignore *big.Int pointers
	for _, index := range bigIntPtrPattern.FindAllIndex(dump, -1) {
		copy(dump[index[0]:index[1]], []byte("(*big.Int)(0x0000000000)"))
	}

	var (
		ptrIndices = ptrPattern.FindAllIndex(dump, -1)
		visitedPtr = make(map[string]int)
		numPtrs    int
	)
	for _, index := range ptrIndices {
		ptrStr := string(dump[index[0]:index[1]])
		if _, ok := visitedPtr[ptrStr]; !ok {
			visitedPtr[ptrStr] = numPtrs
			numPtrs++
		}

		normalized := []byte(fmt.Sprintf("(0x%010x)", visitedPtr[ptrStr]))
		copy(dump[index[0]:index[1]], normalized)
	}
	return string(dump)
}
