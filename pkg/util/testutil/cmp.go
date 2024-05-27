package testutil

import (
	"reflect"

	"github.com/KyberNetwork/int256"
	uniswapcoreentities "github.com/daoleno/uniswap-sdk-core/entities"
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
func CmpOpts() []cmp.Option {
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
