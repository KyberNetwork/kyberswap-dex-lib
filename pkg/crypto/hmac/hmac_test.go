package hmac

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Verify(t *testing.T) {
	type input struct {
		message   []byte
		signature []byte
	}

	hmacSealer := NewHMACSealer([]byte("secret"))
	signature, _ := hmacSealer.Sign([]byte("msg"))
	fakeSignature, _ := hmacSealer.Sign([]byte("msg1"))

	tests := []struct {
		name  string
		input input
		err   error
	}{
		{
			name: "Should verify successfully when data is valid which includes the message and signature",
			input: input{
				signature: signature,
				message:   []byte("msg"),
			},
			err: nil,
		},
		{
			name: "Should return an error when signature is invalid",
			input: input{
				signature: fakeSignature,
				message:   []byte("msg"),
			},
			err: ErrSignatureMisMatch,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := hmacSealer.Verify(tt.input.message, tt.input.signature)
			assert.Equal(t, tt.err, err)
		})
	}
}
