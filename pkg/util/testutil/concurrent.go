package testutil

import (
	"reflect"
	"runtime"
	"sync"
	"testing"

	"github.com/samber/lo"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/israce"
)

var (
	concurrentFactor = runtime.NumCPU() * 10
)

func mustReturnSameOutputAndConcurrentSafe[R any](t *testing.T, f func() (R, error)) (ret R, err error) {
	if concurrentFactor <= 0 {
		panic("n must > 0")
	}

	if !israce.Enabled {
		panic("race detector must be enabled, please run/build with -race options")
	}

	var (
		wg      sync.WaitGroup
		outputs = make([]lo.Tuple2[R, error], concurrentFactor)
	)
	for i := 0; i < concurrentFactor; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			value, err := f()
			outputs[i] = lo.T2(value, err)
		}(i)
	}
	wg.Wait()

	reference := outputs[0]

	for i := 1; i < concurrentFactor; i++ {
		if !reflect.DeepEqual(reference.A, outputs[i].A) {
			t.Fatalf("outputs value are not equal, expected %v actual %v", reference.A, outputs[i].A)
			return
		}

		if (reference.B == nil) != (outputs[i].B == nil) {
			t.Fatalf("outputs are not equal, expected %v actual %v", reference, outputs[i])
			return
		} else if reference.B != nil && reference.B.Error() != outputs[i].B.Error() {
			t.Fatalf("outputs error are not equal, expected %v actual %v", reference.B, outputs[i].B)
			return
		}
	}

	return reference.Unpack()
}

// MustConcurrentSafe check concurrent calls of the same parameters
//
// * are not racy AND
//
// * produces the same output
func MustConcurrentSafe[R any](t *testing.T, f func() (R, error)) (R, error) {
	if israce.Enabled {
		return mustReturnSameOutputAndConcurrentSafe[R](t, f)
	}
	return f()
}
