package helper

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtension(t *testing.T) {
	tests := []struct {
		name    string
		input   ExtensionData
		encoded string
		wantErr bool
	}{
		{
			name: "simple extension",
			input: ExtensionData{
				MakerAssetSuffix: "0x01",
				TakerAssetSuffix: "0x02",
				MakerPermit:      "0x03",
				Predicate:        "0x04",
				MakingAmountData: "0x05",
				TakingAmountData: "0x06",
				PreInteraction:   "0x07",
				PostInteraction:  "0x08",
				CustomData:       "0xff",
			},
			encoded: "0x00000008000000070000000600000005000000040000000300000002000000010102050604030708ff",
		},
		{
			name: "complex extension with permit (1)",
			input: ExtensionData{
				MakerAssetSuffix: "0x",
				TakerAssetSuffix: "0x",
				MakerPermit:      "0x111111111117dc0aa78b770fa6a738034120c30200000000000000000000000054526dd3b396a3233910baf1b8d195bea3b25021000000000000000000000000111111125421ca6dc452d289314280a0f8842a65ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0000000000000000000000000000000000000000000000000000000067a32268000000000000000000000000000000000000000000000000000000000000001c39c337785ca12a775f14455e997a2824d50542c36ae5ffee855ff8960bf5a91431ca6b66f75153ab651221085b7daa5920ff4944cb08be5e1d2bace94ab36edd",
				Predicate:        "0x",
				MakingAmountData: "0x",
				TakingAmountData: "0x",
				PreInteraction:   "0x",
				PostInteraction:  "0x",
				CustomData:       "0x",
			},
			encoded: "0x000000f4000000f4000000f40000000000000000000000000000000000000000111111111117dc0aa78b770fa6a738034120c30200000000000000000000000054526dd3b396a3233910baf1b8d195bea3b25021000000000000000000000000111111125421ca6dc452d289314280a0f8842a65ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0000000000000000000000000000000000000000000000000000000067a32268000000000000000000000000000000000000000000000000000000000000001c39c337785ca12a775f14455e997a2824d50542c36ae5ffee855ff8960bf5a91431ca6b66f75153ab651221085b7daa5920ff4944cb08be5e1d2bace94ab36edd",
		},
		{
			name: "complex extension with permit (2)",
			input: ExtensionData{
				MakerAssetSuffix: "0x",
				TakerAssetSuffix: "0x",
				MakerPermit:      "0x111111111117dc0aa78b770fa6a738034120c3020000000000000000000000009c93896970b1332700f58eb44ffe3ea88f227ca0000000000000000000000000111111125421ca6dc452d289314280a0f8842a65ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0000000000000000000000000000000000000000000000000000000067cfad28000000000000000000000000000000000000000000000000000000000000001c65bbf3251cb7016d0b154fa816fef95b2b30167847ff9d6f3737766c7a9ae2d8121be4807c7d649e07dc063f23a1b47b39d992fe3f3181f07ebd2b02a7aa3775",
				Predicate:        "0x",
				MakingAmountData: "0x",
				TakingAmountData: "0x",
				PreInteraction:   "0x",
				PostInteraction:  "0x",
				CustomData:       "0x",
			},
			encoded: "0x000000f4000000f4000000f40000000000000000000000000000000000000000111111111117dc0aa78b770fa6a738034120c3020000000000000000000000009c93896970b1332700f58eb44ffe3ea88f227ca0000000000000000000000000111111125421ca6dc452d289314280a0f8842a65ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff0000000000000000000000000000000000000000000000000000000067cfad28000000000000000000000000000000000000000000000000000000000000001c65bbf3251cb7016d0b154fa816fef95b2b30167847ff9d6f3737766c7a9ae2d8121be4807c7d649e07dc063f23a1b47b39d992fe3f3181f07ebd2b02a7aa3775",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test NewExtension and Encode
			extension, err := NewExtension(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)

			encoded := extension.Encode()
			assert.Equal(t, tt.encoded, encoded)

			// Test Decode
			decoded, err := DecodeExtension(tt.encoded)
			assert.NoError(t, err)
			assert.Equal(t, extension, decoded)
		})
	}
}
