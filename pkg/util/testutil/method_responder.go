package testutil

import (
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/require"
)

// MethodResponder answers the calls of a MulticallStub by the method their selector names in one of
// its ABIs, whatever the target: Set gives a method's outputs. A call to a method with none, or
// whose selector is in no ABI, fails (a revert). Outputs can change between reads.
type MethodResponder struct {
	t    testing.TB
	abis []abi.ABI

	mu      sync.Mutex
	outputs map[string][]byte
}

func NewMethodResponder(t testing.TB, abis ...abi.ABI) *MethodResponder {
	return &MethodResponder{t: t, abis: abis, outputs: map[string][]byte{}}
}

// Set makes method answer with outputs packed by its ABI.
func (r *MethodResponder) Set(method string, outputs ...any) *MethodResponder {
	for _, a := range r.abis {
		if m, ok := a.Methods[method]; ok {
			packed, err := m.Outputs.Pack(outputs...)
			require.NoError(r.t, err, method)
			r.mu.Lock()
			r.outputs[method] = packed
			r.mu.Unlock()
			return r
		}
	}
	require.FailNow(r.t, "method is in no ABI", method)
	return r
}

// SetRaw makes method answer with packed, the outputs packed by a shape the method's ABI cannot
// pack (for example the same method of another ABI that a tracker also decodes).
func (r *MethodResponder) SetRaw(method string, packed []byte) *MethodResponder {
	r.mu.Lock()
	r.outputs[method] = packed
	r.mu.Unlock()
	return r
}

// Respond is the respond function of NewMulticallStub.
func (r *MethodResponder) Respond(_ common.Address, data []byte) (bool, []byte) {
	if len(data) < 4 {
		return false, nil
	}
	for _, a := range r.abis {
		m, err := a.MethodById(data[:4])
		if err != nil {
			continue
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		out, ok := r.outputs[m.Name]
		return ok, out
	}
	return false, nil
}
