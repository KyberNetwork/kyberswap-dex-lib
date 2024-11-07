package testutil

import (
	"reflect"
	"runtime"
	"sync"
	"testing"

	"github.com/KyberNetwork/kyberswap-dex-lib/pkg/util/israce"
)

var (
	concurrentFactor = runtime.NumCPU() * 10
)

type valueAndError struct {
	value any
	err   error
}

func mustReturnSameOutputAndConcurrentSafe[R any](t *testing.T, f func() (any, error)) (ret R, err error) {
	if concurrentFactor <= 0 {
		panic("n must > 0")
	}

	if !israce.Enabled {
		panic("race detector must be enabled, please run/build with -race options")
	}

	var (
		wg      sync.WaitGroup
		outputs = make([]valueAndError, concurrentFactor)
	)
	for i := 0; i < concurrentFactor; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			value, err := f()
			outputs[i] = valueAndError{value, err}
		}(i)
	}
	wg.Wait()

	reference := outputs[0]

	for i := 1; i < concurrentFactor; i++ {
		if (reference.value == nil) == (outputs[i].value == nil) {
			if !reflect.DeepEqual(outputs[0].value, outputs[i].value) {
				t.Fatalf("outputs value are not equal, expected %v actual %v", reference.value, outputs[i].value)
				return
			}
		} else {
			t.Fatalf("outputs are not equal, expected %v actual %v", reference, outputs[i])
			return
		}

		if (reference.err == nil) == (outputs[i].err == nil) {
			if reference.err != nil && reference.err.Error() != outputs[i].err.Error() {
				t.Fatalf("outputs error are not equal, expected %v actual %v", reference.err, outputs[i].err)
				return
			}
		} else {
			t.Fatalf("outputs are not equal, expected %v actual %v", reference, outputs[i])
			return
		}
	}

	return reference.value.(R), reference.err
}

// MustConcurrentSafe check concurrent calls of the same parameters
//
// * are not racy AND
//
// * produces the same output
func MustConcurrentSafe[R any](t *testing.T, f func() (any, error)) (R, error) {
	if israce.Enabled {
		return mustReturnSameOutputAndConcurrentSafe[R](t, f)
	}

	value, err := f()
	return value.(R), err
}
