package helper

import (
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtension(t *testing.T) {
	t.Run("should encode/decode", func(t *testing.T) {
		extension := Extension{
			MakerAssetSuffix: []byte{},
			TakerAssetSuffix: []byte{},
			MakingAmountData: []byte{
				251, 40, 9, 165, 49, 68, 115, 225, 22, 95, 107, 88, 1, 142, 32,
				237, 143, 7, 184, 64, 0, 12, 205, 0, 0, 54, 117, 103, 128, 51,
				88, 0, 1, 104, 1, 65, 186, 0, 228, 79, 0, 48, 0, 223, 241, 0,
				48, 0, 219, 142, 0, 48, 0, 215, 42, 0, 48, 0, 190, 211, 0, 48,
				0, 165, 56, 0, 24, 0, 154, 249, 0, 60, 0, 12, 205, 0, 36,
			},
			TakingAmountData: []byte{
				251, 40, 9, 165, 49, 68, 115, 225, 22, 95, 107, 88, 1, 142, 32,
				237, 143, 7, 184, 64, 0, 12, 205, 0, 0, 54, 117, 103, 128, 51,
				88, 0, 1, 104, 1, 65, 186, 0, 228, 79, 0, 48, 0, 223, 241, 0,
				48, 0, 219, 142, 0, 48, 0, 215, 42, 0, 48, 0, 190, 211, 0, 48,
				0, 165, 56, 0, 24, 0, 154, 249, 0, 60, 0, 12, 205, 0, 36,
			},
			Predicate:      []byte{},
			MakerPermit:    []byte{},
			PreInteraction: []byte{},
			PostInteraction: []byte{
				251, 40, 9, 165, 49, 68, 115, 225, 22, 95, 107, 88, 1, 142, 32,
				237, 143, 7, 184, 64, 103, 128, 51, 64, 176, 148, 152, 3, 10,
				227, 65, 107, 102, 220, 0, 0, 109, 229, 224, 228, 40, 172, 119,
				29, 119, 181, 0, 0, 51, 159, 181, 116, 189, 197, 103, 99, 249,
				149, 0, 0, 209, 139, 212, 95, 11, 148, 245, 74, 150, 143, 0, 0,
				214, 27, 137, 43, 42, 214, 36, 144, 17, 133, 0, 0, 187, 46, 246,
				187, 26, 48, 190, 126, 230, 190, 0, 0, 173, 225, 149, 103, 187,
				83, 128, 53, 237, 54, 0, 0, 56,
			},
			CustomData: []byte{},
		}

		encodedExtension := extension.Encode()

		decodedExtension, err := DecodeExtension(hexutil.Encode(encodedExtension))
		require.NoError(t, err)

		require.Equal(t, extension, decodedExtension)
	})

	t.Run("decode empty", func(t *testing.T) {
		encodedExtension := "0x"
		e, err := DecodeExtension(encodedExtension)
		require.NoError(t, err)

		assert.True(t, e.IsEmpty())
	})

	t.Run("decode", func(t *testing.T) {
		// nolint: lll
		encodedExtension := "0x000001070000009a0000009a0000009a0000009a0000004d0000000000000000fb2809a5314473e1165f6b58018e20ed8f07b840000ccd00003675678033580001680141ba00e44f003000dff1003000db8e003000d72a003000bed3003000a5380018009af9003c000ccd0024fb2809a5314473e1165f6b58018e20ed8f07b840000ccd00003675678033580001680141ba00e44f003000dff1003000db8e003000d72a003000bed3003000a5380018009af9003c000ccd0024fb2809a5314473e1165f6b58018e20ed8f07b84067803340b09498030ae3416b66dc00006de5e0e428ac771d77b50000339fb574bdc56763f9950000d18bd45f0b94f54a968f0000d61b892b2ad6249011850000bb2ef6bb1a30be7ee6be0000ade19567bb538035ed36000038"
		e, err := DecodeExtension(encodedExtension)
		require.NoError(t, err)

		assert.False(t, e.IsEmpty())
		assert.NotEmpty(t, e.MakingAmountData)
		assert.NotEmpty(t, e.TakingAmountData)
		assert.NotEmpty(t, e.PostInteraction)
	})
}
