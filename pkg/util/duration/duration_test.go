package duration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDuration_MarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		duration Duration
		expected []byte
		err      error
	}{
		{duration: Duration{5 * time.Second}, expected: []byte("\"5s\""), err: nil},
		{duration: Duration{100 * time.Millisecond}, expected: []byte("\"100ms\""), err: nil},
		{duration: Duration{4 * time.Minute}, expected: []byte("\"4m0s\""), err: nil},
		{duration: Duration{4}, expected: []byte("\"4ns\""), err: nil},
	}

	for _, tc := range testCases {
		got, err := tc.duration.MarshalJSON()

		assert.Equal(t, tc.expected, got)
		assert.Equal(t, tc.err, err)
	}
}

func TestDuration_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		input    []byte
		expected Duration
		err      error
	}{
		{input: []byte("\"5s\""), expected: Duration{5 * time.Second}, err: nil},
		{input: []byte("\"100ms\""), expected: Duration{100 * time.Millisecond}, err: nil},
		{input: []byte("\"4m0s\""), expected: Duration{4 * time.Minute}, err: nil},
		{input: []byte("\"4ns\""), expected: Duration{4}, err: nil},
	}

	for _, tc := range testCases {
		var d Duration
		err := d.UnmarshalJSON(tc.input)

		assert.Equal(t, tc.expected, d)
		assert.ErrorIs(t, err, tc.err)
	}
}
