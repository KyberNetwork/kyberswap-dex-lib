package nativev1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuoteParams_ToMap(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args QuoteParams
		want map[string]string
	}{
		{
			"happy",
			QuoteParams{
				SrcChain:           "1",
				DstChain:           "1",
				TokenIn:            "0xdac17f958d2ee523a2206206994597c13d831ec7",
				TokenOut:           "0x6b175474e89094c44da98b954eedeac495271d0f",
				AmountWei:          "1000000000000000000",
				FromAddress:        "0x6b175474e89094c44da98b954eedeac495271d0f",
				BeneficiaryAddress: "0x6b175474e89094c44da98b954eedeac495271d0f",
				ToAddress:          "0x6b175474e89094c44da98b954eedeac495271d0f",
				ExpiryTime:         100,
				Slippage:           "0.1",
			},
			map[string]string{
				"src_chain":           "1",
				"dst_chain":           "1",
				"token_in":            "0xdac17f958d2ee523a2206206994597c13d831ec7",
				"token_out":           "0x6b175474e89094c44da98b954eedeac495271d0f",
				"amount_wei":          "1000000000000000000",
				"from_address":        "0x6b175474e89094c44da98b954eedeac495271d0f",
				"beneficiary_address": "0x6b175474e89094c44da98b954eedeac495271d0f",
				"to_address":          "0x6b175474e89094c44da98b954eedeac495271d0f",
				"expiry_time":         "100",
				"slippage":            "0.1",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equalf(t, tt.want, tt.args.ToMap(), "ToMap()")
		})
	}
}
