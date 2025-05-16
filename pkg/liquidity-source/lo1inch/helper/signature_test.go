package helper

import (
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/stretchr/testify/assert"
)

func Test_parseSignature(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		signature          string
		compactedSignature string
		wantR              string
		wantVS             string
		wantErr            assert.ErrorAssertionFunc
	}{
		{
			name:               "should correct when yParity=27 (0x1b)",
			signature:          "0xb52e7a15e7246dc138635ce15d6318fc29629b18aaea3235bfc1892475eaa70363f4c5236010a9cf9363633d6fcc942d985c26f8be988dba01fe4bd23a2123551b",
			compactedSignature: "0xb52e7a15e7246dc138635ce15d6318fc29629b18aaea3235bfc1892475eaa70363f4c5236010a9cf9363633d6fcc942d985c26f8be988dba01fe4bd23a212355",
			wantR:              "0xb52e7a15e7246dc138635ce15d6318fc29629b18aaea3235bfc1892475eaa703",
			wantVS:             "0x63f4c5236010a9cf9363633d6fcc942d985c26f8be988dba01fe4bd23a212355",
			wantErr:            assert.NoError,
		},
		{
			name:               "should correct when yParity=28 (0x1c)",
			signature:          "0x4b7e9f7985b493945a5a0355bc4c58d2171afdf3d5b9055d45a9bd5a2d1ebafe7a86917cc442741700bf89f00d777a81f47fbefd48cb73b05d96e78eaa0a71da1c",
			compactedSignature: "0x4b7e9f7985b493945a5a0355bc4c58d2171afdf3d5b9055d45a9bd5a2d1ebafefa86917cc442741700bf89f00d777a81f47fbefd48cb73b05d96e78eaa0a71da",
			wantR:              "0x4b7e9f7985b493945a5a0355bc4c58d2171afdf3d5b9055d45a9bd5a2d1ebafe",
			wantVS:             "0xfa86917cc442741700bf89f00d777a81f47fbefd48cb73b05d96e78eaa0a71da",
			wantErr:            assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LO1inchParseSignature(tt.signature)
			if !tt.wantErr(t, err, fmt.Sprintf("parseSignature(%v)", tt.signature)) {
				return
			}

			assert.Equal(t, common.FromHex(tt.wantR), got.R)
			assert.Equal(t, common.FromHex(tt.wantVS), got.yParityAndSBytes())

			compactedSignatureBytes := got.GetCompactedSignatureBytes()
			assert.Equal(t, 64, len(compactedSignatureBytes))
			assert.Equal(t, common.FromHex(tt.compactedSignature), compactedSignatureBytes)
			assert.Equal(t, append(common.FromHex(tt.wantR), common.FromHex(tt.wantVS)...), compactedSignatureBytes)

		})
	}
}

func Test_getNormalizedV(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		v       byte
		want    int
		wantErr bool
	}{
		{
			name: "Test 1",
			v:    0,
			want: 27,
		},
		{
			name: "Test 2",
			v:    1,
			want: 28,
		},
		{
			name: "Test 3",
			v:    28,
			want: 28,
		},
		{
			name: "Test 4",
			v:    27,
			want: 27,
		},
		{
			name: "Test 5",
			v:    46,
			want: 28,
		},
		{
			name: "Test 6",
			v:    45,
			want: 27,
		},
		{
			name:    "Invalid v",
			v:       34,
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getNormalizedV(tt.v)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			assert.Equal(t, tt.want, got)
		})
	}
}
