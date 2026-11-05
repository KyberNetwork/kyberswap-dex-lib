package ephemeral

import (
	"bytes"
	"fmt"
	"math/big"
	"math/rand/v2"
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	gmath "github.com/ethereum/go-ethereum/common/math"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// refCanonical is how UnpackCall told a result before checkCanonical: decode, check every width,
// and encode again, byte for byte.
func refCanonical(args abi.Arguments, payload []byte) bool {
	values, err := args.Unpack(payload)
	if err != nil {
		return false
	}
	for i, arg := range args {
		if refWidths(arg.Type, reflect.ValueOf(values[i]), arg.Name) != nil {
			return false
		}
	}
	again, err := args.Pack(values...)
	return err == nil && bytes.Equal(again, payload)
}

// refWidths is how UnpackCall checked widths before checkCanonical: on the decoded values.
func refWidths(t abi.Type, v reflect.Value, path string) error {
	switch t.T {
	case abi.IntTy, abi.UintTy:
		switch t.Size {
		case 8, 16, 32, 64, 256: // native, or a full word
			return nil
		}
		n, ok := reflect.TypeAssert[*big.Int](v)
		if !ok {
			return fmt.Errorf("%s: %T for %s", path, v.Interface(), t)
		}
		if refFits(n, t.Size, t.T == abi.IntTy) {
			return nil
		}
		return fmt.Errorf("%s: %s does not fit %s", path, n, t)
	case abi.SliceTy, abi.ArrayTy:
		for i := range v.Len() {
			if err := refWidths(*t.Elem, v.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	case abi.TupleTy:
		for i, elem := range t.TupleElems {
			name := path + "." + strings.TrimSpace(t.TupleRawNames[i])
			if err := refWidths(*elem, v.Field(i), name); err != nil {
				return err
			}
		}
	}
	return nil
}

func refFits(n *big.Int, bits int, signed bool) bool {
	if !signed {
		return n.Sign() >= 0 && n.BitLen() <= bits
	}
	limit := new(big.Int).Lsh(big.NewInt(1), uint(bits-1))
	return n.Cmp(limit) < 0 && n.Cmp(new(big.Int).Neg(limit)) >= 0
}

// TestCheckCanonicalAgreesWithEncodingAgain: on results and on results with bytes changed, cut or
// added, checkCanonical takes exactly what decoding and encoding again gives back.
func TestCheckCanonicalAgreesWithEncodingAgain(t *testing.T) {
	type batchResult struct {
		Ran  bool
		Gas  *big.Int
		Data []byte
	}
	results := []batchResult{{true, big.NewInt(5), []byte{1, 2, 3}}, {false, big.NewInt(0), nil},
		{true, big.NewInt(1 << 40), bytes.Repeat([]byte{0xab}, 64)}}
	batch, err := lensBatch.result.Pack(big.NewInt(100), big.NewInt(3), results)
	require.NoError(t, err)
	emptyBatch, err := lensBatch.result.Pack(big.NewInt(100), big.NewInt(0), []batchResult{})
	require.NoError(t, err)
	tick := common32(7)
	for _, c := range []struct {
		args    abi.Arguments
		payload []byte
	}{
		{testContract.result, append(bytes.Clone(tick), tickWord(-5)...)},
		{testContract.result, append(bytes.Clone(tick), tickWord(1<<23-1)...)},
		{testContract.result, append(bytes.Clone(tick), tickWord(-1<<23)...)},
		{lensBatch.result, batch},
		{lensBatch.result, emptyBatch},
	} {
		require.NoError(t, checkCanonical(c.args, c.payload))
		require.True(t, refCanonical(c.args, c.payload))

		rng := rand.New(rand.NewPCG(1, uint64(len(c.payload))))
		accepted := 0
		for i := range 20000 {
			p := mutate(rng, c.payload)
			got, want := checkCanonical(c.args, p) == nil, refCanonical(c.args, p)
			require.Equal(t, want, got, "mutation %d: %x", i, p)
			if got {
				accepted++
			}
		}
		// Some changes leave a result, of other values: both sides must be met.
		assert.Positive(t, accepted)
		assert.Less(t, accepted, 20000)
	}
}

func tickWord(v int64) []byte {
	return gmath.U256Bytes(big.NewInt(v))
}

// mutate changes p as a passed-up revert or a broken node might: a byte or a word changed, often
// one that is an offset or a length, words cut or added.
func mutate(rng *rand.Rand, p []byte) []byte {
	p = bytes.Clone(p)
	switch rng.IntN(6) {
	case 0:
		if len(p) > 0 {
			p[rng.IntN(len(p))] ^= byte(1 + rng.IntN(255))
		}
	case 1: // a small change to the low byte of a word: an offset, a length, a bool, a width
		if len(p) >= 32 {
			w := rng.IntN(len(p)/32) * 32
			p[w+31] += byte(rng.IntN(5)) - 2
		}
	case 2: // a high byte of a word: a value past its width, or a sign
		if len(p) >= 32 {
			w := rng.IntN(len(p)/32) * 32
			p[w+rng.IntN(29)] ^= 0x80 >> rng.IntN(8)
		}
	case 3:
		if len(p) >= 32 {
			p = p[:len(p)-32*(1+rng.IntN(min(3, len(p)/32)))]
		}
	case 4:
		p = append(p, make([]byte, 32)...)
	case 5: // a word set to all ones or a length past the payload
		if len(p) >= 32 {
			w := rng.IntN(len(p)/32) * 32
			fill := byte(0xff * rng.IntN(2))
			for i := range 32 {
				p[w+i] = fill
			}
			if fill == 0 {
				p[w+30] = 1
			}
		}
	}
	return p
}
