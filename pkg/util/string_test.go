package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatKey(t *testing.T) {
	tests := []struct {
		sep    string
		args   []string
		expect string
	}{
		{
			sep:    ":",
			args:   []string{"prefix-key", "123456"},
			expect: "prefix-key:123456",
		},
		{
			sep:    "-",
			args:   []string{"prefix_key", "654321"},
			expect: "prefix_key-654321",
		},
	}

	for _, test := range tests {
		key := FormatKey(test.sep, test.args...)
		assert.Equal(t, test.expect, key)
	}
}
